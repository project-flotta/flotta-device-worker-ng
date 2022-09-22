package resources

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	cgroupsv2 "github.com/containerd/cgroups/v2"
	"github.com/tupyy/device-worker-ng/internal/entity"
	"go.uber.org/zap"
)

var (
	ErrCPUMaxFileNotFound = errors.New("cpu.max file not found")
)

type SliceType int

const (
	UserSlice SliceType = iota
	SystemSlice
	MachineSlice
)

type ResourceManager struct {
}

func New() *ResourceManager {
	return &ResourceManager{}
}

func (r *ResourceManager) SliceExists(ctx context.Context, path string) bool {
	_, err := cgroupsv2.LoadSystemd("/", path)
	if err != nil {
		return false
	}
	return true
}

func (r *ResourceManager) CreateSlice(ctx context.Context, w entity.Workload) error {
	sliceType := MachineSlice
	if w.IsRootless() {
		sliceType = UserSlice
	}
	rootPath := r.getRootSlice(sliceType)
	cgroupPath := path.Join(rootPath, "flotta.slice", fmt.Sprintf("flotta-%s.slice", strings.ReplaceAll(w.ID(), "-", "_")))
	if err := os.MkdirAll(path.Join("/sys/fs/cgroup", cgroupPath), 0777); err != nil {
		return fmt.Errorf("failed to create cgroup for workload '%s': %w", w.ID(), err)
	}
	// toggle controllers
	m, err := cgroupsv2.LoadManager("/sys/fs/cgroup", cgroupPath)
	if err != nil {
		return err
	}
	return m.ToggleControllers([]string{"cpu", "memory"}, cgroupsv2.Enable)
}

func (r *ResourceManager) RemoveSlice(ctx context.Context, path string) error {
	m, err := cgroupsv2.LoadSystemd("/", path)
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
	m, err := cgroupsv2.LoadManager("/sys/fs/cgroup", path)
	if err != nil {
		return err
	}
	quota := int64(resources.Value1)
	period := resources.Value2
	cpuMax := cgroupsv2.NewCPUMax(&quota, &period)
	if err := m.Update(&cgroupsv2.Resources{
		CPU: &cgroupsv2.CPU{
			Max: cpuMax,
		},
	}); err != nil {
		zap.S().Errorw("failed to set resources", "path", path, "error", err)
		return err
	}

	zap.S().Debugw("resources set", "cgroup", path, "cpu", cpuMax, "error", err)
	return nil
}

func (r *ResourceManager) GetResources(ctx context.Context, cgroup string) (entity.CpuResource, error) {
	_, err := cgroupsv2.LoadManager("/sys/fs/cgroup", cgroup)
	if err != nil {
		return entity.CpuResource{}, fmt.Errorf("failed to read cgroup '%s': '%s'", cgroup, err)
	}

	cpuFilePath := path.Join("/sys/fs/cgroup", cgroup, "cpu.max")
	f, err := os.Open(cpuFilePath)
	if err != nil {
		return entity.CpuResource{}, fmt.Errorf("%w failed to open file '%s': '%s'", ErrCPUMaxFileNotFound, cpuFilePath, err)
	}
	defer f.Close()

	cpu := entity.CpuResource{Value2: 100000}
	s := bufio.NewScanner(f)
	for s.Scan() {
		parts := strings.Split(s.Text(), " ")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				if part != "max" {
					val, err := strconv.Atoi(part)
					if err != nil {
						return entity.CpuResource{}, err
					}
					cpu.Value1 = uint64(val)
				} else {
					cpu.Value1 = uint64(100000)
				}
				return cpu, nil
			}
		}
	}

	return entity.CpuResource{}, nil

}

func (r *ResourceManager) GetRootSlice(sliceType SliceType) string {
	var sliceRegex *regexp.Regexp
	switch sliceType {
	case UserSlice:
		sliceRegex = regexp.MustCompile("user\\.slice/user-\\d*\\.slice")
	case MachineSlice:
		return "machine.slice"
	case SystemSlice:
		return "system.slice"
	}

	cgroup, err := r.GetCGroup(context.TODO(), sliceRegex, false)
	if err != nil {
		zap.S().Errorw("failed to get slice", "error", err)
		return ""
	}
	return cgroup
}

func (r *ResourceManager) GetCGroup(ctx context.Context, rxp *regexp.Regexp, fullPath bool) (string, error) {
	searchFn := func(ctx context.Context, root string, regex *regexp.Regexp, output chan string, errCh chan error, fullPath bool) {
		err := filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && regex.MatchString(info.Name()) {
				if fullPath {
					output <- path
				} else {
					output <- info.Name()
				}
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			return nil
		})
		errCh <- err
	}

	result := make(chan string, 1)
	errCh := make(chan error, 1)
	searchCtx, cancel := context.WithCancel(ctx)

	go searchFn(searchCtx, "/sys/fs/cgroup/", rxp, result, errCh, fullPath)

	select {
	case cgroup := <-result:
		cancel()
		return cgroup, nil
	case err := <-errCh:
		zap.S().Errorw("error during cgroup search", "error", err)
	}

	cancel()
	return "", errors.New("failed to found cgroup")

}
func (r *ResourceManager) createSlice(ctx context.Context, path string) error {
	res := cgroupsv2.Resources{
		CPU: &cgroupsv2.CPU{
			Max:  "max 100000",
			Cpus: "0",
			Mems: "0",
		},
	}
	_, err := cgroupsv2.NewSystemd("/", path, -1, &res)
	if err != nil {
		return err
	}
	return nil
}

func (r *ResourceManager) getRootSlice(sliceType SliceType) string {
	switch sliceType {
	case UserSlice:
		return "/user.slice/user-1000.slice/user@1000.service"
	case SystemSlice:
		return "/system.slice"
	case MachineSlice:
		return "/machine.slice"
	}
	return ""
}
