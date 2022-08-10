package edge

import (
	"context"
	"fmt"
	"time"

	config "github.com/tupyy/device-worker-ng/configuration"
	"github.com/tupyy/device-worker-ng/internal/certificate"
	"github.com/tupyy/device-worker-ng/internal/configuration"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -package=edge -destination=mock_client.go --build_flags=--mod=mod . Client
type Client interface {
	// Enrol sends the enrolment information.
	Enrol(ctx context.Context, deviceID string, info entity.EnrolementInfo) error

	// Register sends the registration info.
	// Registration info is actually a csr which will be signed by the operator and send back with the response.
	Register(ctx context.Context, deviceID string, registerInfo entity.RegistrationInfo) (entity.RegistrationResponse, error)

	// Heartbeat
	Heartbeat(ctx context.Context, deviceID string, heartbeat entity.Heartbeat) error

	// GetConfiguration get the configuration from flotta-operator
	GetConfiguration(ctx context.Context, deviceID string) (entity.DeviceConfigurationMessage, error)
}

type Controller struct {
	client      Client
	confManager *configuration.Manager
	certManager *certificate.Manager
	done        chan chan struct{}
}

func New(client Client, confManager *configuration.Manager, certManager *certificate.Manager) *Controller {
	c := &Controller{
		client:      client,
		confManager: confManager,
		certManager: certManager,
		done:        make(chan chan struct{}, 1),
	}

	go c.run()

	return c
}

func (c *Controller) Shutdown() {
	d := make(chan struct{}, 1)
	c.done <- d
	<-d
}

func (c *Controller) run() {
	var (
		register      chan struct{}
		enrol         = make(chan struct{}, 1)
		op            = make(chan struct{}, 1)
		configuration = make(chan time.Duration, 1)
	)

	ticker := time.NewTicker(c.confManager.Configuration().Configuration.Heartbeat.Period)

	for {
		select {
		case <-enrol:
			zap.S().Info("Enrolling device")

			enrolInfo := entity.EnrolementInfo{
				Features: entity.EnrolmentInfoFeatures{
					Hardware: c.confManager.HardwareInfo(),
				},
				TargetNamespace: config.GetTargetNamespace(),
			}

			if err := c.client.Enrol(context.TODO(), config.GetDeviceID(), enrolInfo); err != nil {
				zap.S().Errorw("Cannot enroll device", "error", err, "enrolement info", enrolInfo)
				break
			}

			enrol = nil
			register = make(chan struct{}, 1)

			zap.S().Info("Device enrolled")
		case <-register:
			zap.S().Info("Registering device")

			csr, key, err := c.certManager.GenerateCSR("deviceID")
			if err != nil {
				zap.S().Errorw("Cannot generate CSR for registration", "error", err)
				break
			}

			registerInfo := entity.RegistrationInfo{
				CertificateRequest: string(csr),
				Hardware:           c.confManager.HardwareInfo(),
			}

			res, err := c.client.Register(context.TODO(), config.GetDeviceID(), registerInfo)
			if err != nil {
				zap.S().Errorw("Cannot register device", "error", err, "registration info", registerInfo)
				break
			}

			c.certManager.SetCertificate(res.SignedCSR, key)

			if err := c.certManager.WriteCertificate(config.GetCertificateFile(), config.GetPrivateKey()); err != nil {
				zap.S().Errorw("cannot write certificates", "error", err)
				break
			}

			// registration has been successful
			register = nil

			zap.S().Info("Device registered")
		case <-op:
			// This branch handles the main operations: send heartbeat and get the configuration.
			// If there is an error of type UnauthorizedAccessError restart the registration process.
			// For any other error, we keep this branch active.
			// TODO in case of an error other than 401, replace the ticker with a back-off retry

			// We execute _heartbeat_ and _configuration_ op asynchronously but
			// we stop at the first error.
			g, ctx := errgroup.WithContext(context.Background())

			g.Go(func() error {
				err := c.client.Heartbeat(ctx, config.GetDeviceID(), c.confManager.Heartbeat())
				if err != nil {
					return fmt.Errorf("cannot send heartbeat: '%w'", err)
				}

				return nil
			})

			g.Go(func() error {
				configurationMessage, err := c.client.GetConfiguration(ctx, config.GetDeviceID())
				if err != nil {
					return fmt.Errorf("cannot get configuration '%w'", err)
				}

				// reset the ticker if the heartbeat period changed.
				if configurationMessage.Configuration.Heartbeat.Period != c.confManager.Configuration().Configuration.Heartbeat.Period {
					zap.S().Infof("new heartbeat period: %s", configurationMessage.Configuration.Heartbeat.Period)
					configuration <- configurationMessage.Configuration.Heartbeat.Period
				}

				c.confManager.SetConfiguration(configurationMessage)

				return nil
			})

			if err := g.Wait(); err != nil {
				zap.S().Errorf("Error during op: %s", err)

				// TODO refactor this into something better
				switch err.(type) {
				case UnauthorizedAccessError:
					// start the registration process once again
					enrol = make(chan struct{}, 1)
				default:
					// it is something with code != 401 so we keep going doing op
				}
			}
		case heartbeatPeriod := <-configuration:
			// this branch reset the ticker when a new configuration period is set
			ticker.Reset(heartbeatPeriod)
		case <-ticker.C:
			// if enrol or registration channels are not nil then start the enrol and registration process.
			// Otherwise process directly with normal operation
			if enrol != nil {
				enrol <- struct{}{}
				break
			}

			if register != nil {
				register <- struct{}{}
				break
			}

			op <- struct{}{}
		case d := <-c.done:
			zap.S().Info("shutdown controller")
			ticker.Stop()
			d <- struct{}{}
		}
	}
}
