package edge

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	config "github.com/tupyy/device-worker-ng/configuration"
	"github.com/tupyy/device-worker-ng/internal/certificate"
	"github.com/tupyy/device-worker-ng/internal/configuration"
	"github.com/tupyy/device-worker-ng/internal/entities"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -package=edge -destination=mock_client.go --build_flags=--mod=mod . Client
type Client interface {
	// UpdateTLS updates the tls configuration of the client.
	// Updating TLS configuration is required after a successful registration when the CSR is signed by the operator and
	// it must be used to be able to connect to the cluster.
	UpdateTLS(newTlS *tls.Config)

	// Enrol sends the enrolment information.
	Enrol(ctx context.Context, info entities.EnrolementInfo) error

	// Register sends the registration info.
	// Registration info is actually a csr which will be signed by the operator and send back with the response.
	Register(ctx context.Context, registerInfo entities.RegistrationInfo) ([]byte, error)

	// Heartbeat
	Heartbeat(ctx context.Context, heartbeat entities.Heartbeat) error

	// GetConfiguration get the configuration from flotta-operator
	GetConfiguration(ctx context.Context) (entities.DeviceConfiguration, error)
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

	ticker := time.NewTicker(c.confManager.Configuration().Heartbeat.Period)

	for {
		select {
		case <-enrol:
			zap.S().Info("Enrolling device")

			enrolInfo := entities.EnrolementInfo{
				Features: entities.EnrolmentInfoFeatures{
					Hardware: c.confManager.GetHardwareInfo(),
				},
				TargetNamespace: config.GetTargetNamespace(),
			}

			if err := c.client.Enrol(context.TODO(), enrolInfo); err != nil {
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

			registerInfo := entities.RegistrationInfo{
				CertificateRequest: string(csr),
				Hardware:           c.confManager.GetHardwareInfo(),
			}

			signedCSR, err := c.client.Register(context.TODO(), registerInfo)
			if err != nil {
				zap.S().Errorw("Cannot register device", "error", err, "registration info", registerInfo)
				break
			}

			c.certManager.SetCertificate(signedCSR, key)

			if err := c.certManager.WriteCertificate(config.GetCertificateFile(), config.GetPrivateKey()); err != nil {
				zap.S().Errorw("cannot write certificates", "error", err)
				break
			}

			newTLS, err := c.certManager.TLSConfig()
			if err != nil {
				zap.S().Error("cannot create the tls config from signed CSR")
				break
			}

			// update tls config of the client
			c.client.UpdateTLS(newTLS)

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
				err := c.client.Heartbeat(ctx, c.confManager.Heartbeat())
				if err != nil {
					return fmt.Errorf("cannot send heartbeat: '%w'", err)
				}

				return nil
			})

			g.Go(func() error {
				newConfiguration, err := c.client.GetConfiguration(ctx)
				if err != nil {
					return fmt.Errorf("cannot get configuration '%w'", err)
				}

				// reset the ticker if the heartbeat period changed.
				if newConfiguration.Heartbeat.Period != c.confManager.Configuration().Heartbeat.Period {
					zap.S().Infof("new heartbeat period: %s", newConfiguration.Heartbeat.Period)
					configuration <- newConfiguration.Heartbeat.Period
				}

				c.confManager.SetConfiguration(newConfiguration)

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
