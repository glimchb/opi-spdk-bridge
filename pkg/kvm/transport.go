// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package kvm automates plugging of SPDK devices to a QEMU instance
package kvm

import (
	"errors"
	"log"
	"os"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"github.com/opiproject/opi-spdk-bridge/pkg/frontend"
	"github.com/opiproject/opi-spdk-bridge/pkg/utils"
)

type nvmeVfiouserTransport struct {
	ctrlrDir string
}

// NewNvmeVfiouserTransport creates a new instance of nvmeVfiouserTransport
func NewNvmeVfiouserTransport(ctrlrDir string) frontend.NvmeTransport {
	if ctrlrDir == "" {
		log.Panicf("ctrlrDir cannot be empty")
	}

	dir, err := os.Stat(ctrlrDir)
	if err != nil {
		log.Panicf("%v path cannot be evaluated", ctrlrDir)
	}
	if !dir.IsDir() {
		log.Panicf("%v is not a directory", ctrlrDir)
	}

	return &nvmeVfiouserTransport{
		ctrlrDir: ctrlrDir,
	}
}

func (c *nvmeVfiouserTransport) Params(ctrlr *pb.NvmeController, subsys *pb.NvmeSubsystem) (spdk.NvmfSubsystemAddListenerParams, error) {
	pcieID := ctrlr.GetSpec().GetPcieId()
	if pcieID.PortId.Value != 0 {
		return spdk.NvmfSubsystemAddListenerParams{},
			errors.New("only port 0 is supported for vfiouser")
	}

	if pcieID.VirtualFunction.Value != 0 {
		return spdk.NvmfSubsystemAddListenerParams{},
			errors.New("virtual functions are not supported for vfiouser")
	}

	if subsys.Spec.Hostnqn != "" {
		return spdk.NvmfSubsystemAddListenerParams{},
			errors.New("hostnqn for subsystem is not supported for vfiouser")
	}

	result := spdk.NvmfSubsystemAddListenerParams{}
	ctrlrDirPath := controllerDirPath(c.ctrlrDir, utils.GetSubsystemIDFromNvmeName(ctrlr.Name))
	result.Nqn = subsys.Spec.Nqn
	result.ListenAddress.Trtype = "vfiouser"
	result.ListenAddress.Traddr = ctrlrDirPath

	return result, nil
}
