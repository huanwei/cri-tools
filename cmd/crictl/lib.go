package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/containers/storage/pkg/config"
	"github.com/containers/storage/pkg/idtools"
	"io/ioutil"
	"os"
	"strings"
)

// StoreOptions is used for passing initialization options to GetStore(), for
// initializing a Store object and the underlying storage that it controls.
type StoreOptions struct {
	// RunRoot is the filesystem path under which we can store run-time
	// information, such as the locations of active mount points, that we
	// want to lose if the host is rebooted.
	RunRoot string `json:"runroot,omitempty"`
	// GraphRoot is the filesystem path under which we will store the
	// contents of layers, images, and containers.
	GraphRoot string `json:"root,omitempty"`
	// GraphDriverName is the underlying storage driver that we'll be
	// using.  It only needs to be specified the first time a Store is
	// initialized for a given RunRoot and GraphRoot.
	GraphDriverName string `json:"driver,omitempty"`
	// GraphDriverOptions are driver-specific options.
	GraphDriverOptions []string `json:"driver-options,omitempty"`
	// UIDMap and GIDMap are used for setting up a container's root filesystem
	// for use inside of a user namespace where UID mapping is being used.
	UIDMap []idtools.IDMap `json:"uidmap,omitempty"`
	GIDMap []idtools.IDMap `json:"gidmap,omitempty"`
}

// TOML-friendly explicit tables used for conversions.
type tomlConfig struct {
	Storage struct {
		Driver    string                         `toml:"driver"`
		RunRoot   string                         `toml:"runroot"`
		GraphRoot string                         `toml:"graphroot"`
		Options   struct{ config.OptionsConfig } `toml:"options"`
	} `toml:"storage"`
}

// ReloadConfigurationFile parses the specified configuration file and overrides
// the configuration in storeOptions.
func ReloadConfigurationFile(configFile string, storeOptions *StoreOptions) {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Failed to read %s %v\n", configFile, err.Error())
			return
		}
	}

	config := new(tomlConfig)

	if _, err := toml.Decode(string(data), config); err != nil {
		fmt.Printf("Failed to parse %s %v\n", configFile, err.Error())
		return
	}
	if config.Storage.Driver != "" {
		storeOptions.GraphDriverName = config.Storage.Driver
	}
	if config.Storage.RunRoot != "" {
		storeOptions.RunRoot = config.Storage.RunRoot
	}
	if config.Storage.GraphRoot != "" {
		storeOptions.GraphRoot = config.Storage.GraphRoot
	}
	if config.Storage.Options.Thinpool.AutoExtendPercent != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.thinp_autoextend_percent=%s", config.Storage.Options.Thinpool.AutoExtendPercent))
	}

	if config.Storage.Options.Thinpool.AutoExtendThreshold != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.thinp_autoextend_threshold=%s", config.Storage.Options.Thinpool.AutoExtendThreshold))
	}

	if config.Storage.Options.Thinpool.BaseSize != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.basesize=%s", config.Storage.Options.Thinpool.BaseSize))
	}
	if config.Storage.Options.Thinpool.BlockSize != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.blocksize=%s", config.Storage.Options.Thinpool.BlockSize))
	}
	if config.Storage.Options.Thinpool.DirectLvmDevice != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.directlvm_device=%s", config.Storage.Options.Thinpool.DirectLvmDevice))
	}
	if config.Storage.Options.Thinpool.DirectLvmDeviceForce != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.directlvm_device_force=%s", config.Storage.Options.Thinpool.DirectLvmDeviceForce))
	}
	if config.Storage.Options.Thinpool.Fs != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.fs=%s", config.Storage.Options.Thinpool.Fs))
	}
	if config.Storage.Options.Thinpool.LogLevel != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.libdm_log_level=%s", config.Storage.Options.Thinpool.LogLevel))
	}
	if config.Storage.Options.Thinpool.MinFreeSpace != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.min_free_space=%s", config.Storage.Options.Thinpool.MinFreeSpace))
	}
	if config.Storage.Options.Thinpool.MkfsArg != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.mkfsarg=%s", config.Storage.Options.Thinpool.MkfsArg))
	}
	if config.Storage.Options.Thinpool.MountOpt != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.mountopt=%s", config.Storage.Driver, config.Storage.Options.Thinpool.MountOpt))
	}
	if config.Storage.Options.Thinpool.UseDeferredDeletion != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.use_deferred_deletion=%s", config.Storage.Options.Thinpool.UseDeferredDeletion))
	}
	if config.Storage.Options.Thinpool.UseDeferredRemoval != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.use_deferred_removal=%s", config.Storage.Options.Thinpool.UseDeferredRemoval))
	}
	if config.Storage.Options.Thinpool.XfsNoSpaceMaxRetries != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("dm.xfs_nospace_max_retries=%s", config.Storage.Options.Thinpool.XfsNoSpaceMaxRetries))
	}
	for _, s := range config.Storage.Options.AdditionalImageStores {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.imagestore=%s", config.Storage.Driver, s))
	}
	if config.Storage.Options.Size != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.size=%s", config.Storage.Driver, config.Storage.Options.Size))
	}
	if config.Storage.Options.OstreeRepo != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.ostree_repo=%s", config.Storage.Driver, config.Storage.Options.OstreeRepo))
	}
	if config.Storage.Options.SkipMountHome != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.skip_mount_home=%s", config.Storage.Driver, config.Storage.Options.SkipMountHome))
	}
	if config.Storage.Options.MountProgram != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.mount_program=%s", config.Storage.Driver, config.Storage.Options.MountProgram))
	}
	if config.Storage.Options.IgnoreChownErrors != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.ignore_chown_errors=%s", config.Storage.Driver, config.Storage.Options.IgnoreChownErrors))
	}
	if config.Storage.Options.MountOpt != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, fmt.Sprintf("%s.mountopt=%s", config.Storage.Driver, config.Storage.Options.MountOpt))
	}
	if config.Storage.Options.RemapUser != "" && config.Storage.Options.RemapGroup == "" {
		config.Storage.Options.RemapGroup = config.Storage.Options.RemapUser
	}
	if config.Storage.Options.RemapGroup != "" && config.Storage.Options.RemapUser == "" {
		config.Storage.Options.RemapUser = config.Storage.Options.RemapGroup
	}
	if config.Storage.Options.RemapUser != "" && config.Storage.Options.RemapGroup != "" {
		mappings, err := idtools.NewIDMappings(config.Storage.Options.RemapUser, config.Storage.Options.RemapGroup)
		if err != nil {
			fmt.Printf("Error initializing ID mappings for %s:%s %v\n", config.Storage.Options.RemapUser, config.Storage.Options.RemapGroup, err)
			return
		}
		storeOptions.UIDMap = mappings.UIDs()
		storeOptions.GIDMap = mappings.GIDs()
	}

	uidmap, err := idtools.ParseIDMap([]string{config.Storage.Options.RemapUIDs}, "remap-uids")
	if err != nil {
		fmt.Print(err)
	} else {
		storeOptions.UIDMap = append(storeOptions.UIDMap, uidmap...)
	}
	gidmap, err := idtools.ParseIDMap([]string{config.Storage.Options.RemapGIDs}, "remap-gids")
	if err != nil {
		fmt.Print(err)
	} else {
		storeOptions.GIDMap = append(storeOptions.GIDMap, gidmap...)
	}
	if os.Getenv("STORAGE_DRIVER") != "" {
		storeOptions.GraphDriverName = os.Getenv("STORAGE_DRIVER")
	}
	if os.Getenv("STORAGE_OPTS") != "" {
		storeOptions.GraphDriverOptions = append(storeOptions.GraphDriverOptions, strings.Split(os.Getenv("STORAGE_OPTS"), ",")...)
	}
	if len(storeOptions.GraphDriverOptions) == 1 && storeOptions.GraphDriverOptions[0] == "" {
		storeOptions.GraphDriverOptions = nil
	}
}
