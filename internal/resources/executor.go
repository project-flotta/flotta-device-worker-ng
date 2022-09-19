package resources

import (
	"context"
	"fmt"

	cgroupsv2 "github.com/containerd/cgroups/v2"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

type ResourceManager struct {
	rootSlice string
}

func New(rootSlice string) (*ResourceManager, error) {
	r := &ResourceManager{rootSlice: fmt.Sprintf("system-%s", rootSlice)}
	// create the root slice for device-worker-ng
	// the root slice is flotta.slice
	slice := fmt.Sprintf("%s.slice", r.rootSlice)
	if !r.SliceExists(context.TODO(), slice) {
		if err := r.CreateSlice(context.TODO(), slice); err != nil {
			zap.S().Errorw("failed to create root slice", "slice", rootSlice, "error", err)
			return nil, err
		}
	}
	return r, nil
}

func (r *ResourceManager) SliceExists(ctx context.Context, path string) bool {
	_, err := cgroupsv2.LoadSystemd("/", r.createSlicePath(path))
	if err != nil {
		return false
	}
	return true
}

func (r *ResourceManager) CreateSlice(ctx context.Context, path string) error {
	res := cgroupsv2.Resources{}
	_, err := cgroupsv2.NewSystemd("/", r.createSlicePath(path), -1, &res)
	if err != nil {
		return err
	}
	return nil
}

func (r *ResourceManager) RemoveSlice(ctx context.Context, path string) error {
	m, err := cgroupsv2.LoadSystemd("/", r.createSlicePath(path))
	if err != nil {
		return err
	}
	err = m.DeleteSystemd()
	if err != nil {
		return err
	}
	return nil
}

func (r *ResourceManager) Set(ctx context.Context, path string, resources entity.CpuResource) error {
	// TODO
	zap.S().Debug("set resources")
	return nil
}

func (r *ResourceManager) createSlicePath(path string) string {
	return fmt.Sprintf("%s-%s.slice", r.rootSlice, path)
}
