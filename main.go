// Copyright 2012 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"regexp"

	"github.com/hanwen/go-mtpfs/mtp"
	"github.com/hanwen/go-fuse/fuse"
)

func main() {
	fsdebug := flag.Bool("fs-debug", false, "switch on FS debugging")
	mtpDebug := flag.Bool("mtp-debug", false, "switch on MTP debugging")
	vfat := flag.Bool("vfat", true, "assume removable RAM media uses VFAT, and rewrite names.")
	other := flag.Bool("allow-other", false, "allow other users to access mounted fuse. Default: false.")
	deviceFilter := flag.String("dev", "", "regular expression to filter devices.")
	storageFilter := flag.String("storage", "", "regular expression to filter storage areas.")
	flag.Parse()

	if len(flag.Args()) != 1 {
		log.Fatalf("Usage: %s [options] MOUNT-POINT\n", os.Args[0])
	}
	mountpoint := flag.Arg(0)

	dev, err := mtp.SelectDevice(*deviceFilter)
	if err != nil {
		log.Fatalf("detect failed: %v", err)
	}
	
	defer dev.Close()

	if err = dev.OpenSession(); err != nil {
		log.Fatalf("OpenSession failed: %v", err)
	}
	
	sids, err := selectStorages(dev, *storageFilter)
	if err != nil {
		log.Fatalf("selectStorages failed: %v", err)
	}

	dev.DebugPrint = *mtpDebug
	opts := DeviceFsOptions{
		RemovableVFat: *vfat,
	}
	fs, err  := NewDeviceFs(dev, sids, opts)
	if err != nil {
		log.Fatalf("NewDeviceFs failed: %v", err)
	}
	conn := fuse.NewFileSystemConnector(fs, fuse.NewFileSystemOptions())
	rawFs := fuse.NewLockingRawFileSystem(conn)

	mount := fuse.NewMountState(rawFs)
	mOpts := &fuse.MountOptions{
		AllowOther: *other,
	}
	if err := mount.Mount(mountpoint, mOpts); err != nil {
		log.Fatalf("mount failed: %v", err)
	}

	conn.Debug = *fsdebug
	mount.Debug = *fsdebug
	log.Printf("starting FUSE %v", fuse.Version())
	mount.Loop()
}

func selectStorages(dev *mtp.Device, pat string) ([]uint32, error) {
	sids := mtp.Uint32Array{}
	err := dev.GetStorageIDs(&sids)
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pat)
	if err != nil {
		return nil, err
	}

	filtered := []uint32{}
	for _, id := range sids.Values {
		var s mtp.StorageInfo
		err := dev.GetStorageInfo(id, &s)
		if err != nil {
			return nil, err
		}
		
		if !s.IsHierarchical() {
			log.Printf("skipping non hierarchical storage %q", s.StorageDescription)
			continue
		}

		if re.FindStringIndex(s.StorageDescription) == nil {
			log.Printf("filtering out storage %q", s.StorageDescription)
			continue
		}
		filtered = append(filtered, id)
	}

	return filtered, nil
}
