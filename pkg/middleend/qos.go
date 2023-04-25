// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2023 Intel Corporation

// Package middleend implements the MiddleEnd APIs (service) of the storage Server
package middleend

import (
	"context"
	"fmt"
	"log"

	"github.com/opiproject/gospdk/spdk"
	pb "github.com/opiproject/opi-api/storage/v1alpha1/gen/go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// CreateQosVolume creates a QoS volume
func (s *Server) CreateQosVolume(_ context.Context, in *pb.CreateQosVolumeRequest) (*pb.QosVolume, error) {
	log.Printf("CreateQosVolume: Received from client: %v", in)
	if err := s.verifyQosVolume(in.QosVolume); err != nil {
		log.Println("error:", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if volume, ok := s.volumes.qosVolumes[in.QosVolume.QosVolumeId.Value]; ok {
		log.Printf("Already existing QoS volume with id %v", in.QosVolume.QosVolumeId.Value)
		return volume, nil
	}

	params := spdk.BdevQoSParams{
		Name:           in.QosVolume.VolumeId.Value,
		RwIosPerSec:    int(in.QosVolume.LimitMax.RwIopsKiops * 1000),
		RwMbytesPerSec: int(in.QosVolume.LimitMax.RwBandwidthMbs),
		RMbytesPerSec:  int(in.QosVolume.LimitMax.RdBandwidthMbs),
		WMbytesPerSec:  int(in.QosVolume.LimitMax.RdBandwidthMbs),
	}
	var result spdk.BdevQoSResult
	err := s.rpc.Call("bdev_set_qos_limit", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, spdk.ErrFailedSpdkCall
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not set QoS limit: %s", in.QosVolume)
		log.Print(msg)
		return nil, spdk.ErrUnexpectedSpdkCallResult
	}

	s.volumes.qosVolumes[in.QosVolume.QosVolumeId.Value] = proto.Clone(in.QosVolume).(*pb.QosVolume)
	return in.QosVolume, nil
}

// DeleteQosVolume deletes a QoS volume
func (s *Server) DeleteQosVolume(_ context.Context, in *pb.DeleteQosVolumeRequest) (*emptypb.Empty, error) {
	log.Printf("DeleteQosVolume: Received from client: %v", in)
	qosVolume, ok := s.volumes.qosVolumes[in.Name]
	if !ok {
		if in.AllowMissing {
			return &emptypb.Empty{}, nil
		}
		err := status.Errorf(codes.NotFound, "unable to find key %s", in.Name)
		log.Printf("error: %v", err)
		return nil, err
	}
	params := spdk.BdevQoSParams{
		Name:           qosVolume.VolumeId.Value,
		RwIosPerSec:    0,
		RwMbytesPerSec: 0,
		RMbytesPerSec:  0,
		WMbytesPerSec:  0,
	}
	var result spdk.BdevQoSResult
	err := s.rpc.Call("bdev_set_qos_limit", &params, &result)
	if err != nil {
		log.Printf("error: %v", err)
		return nil, spdk.ErrFailedSpdkCall
	}
	log.Printf("Received from SPDK: %v", result)
	if !result {
		msg := fmt.Sprintf("Could not clear QoS limit for : %s", in.Name)
		log.Print(msg)
		return nil, spdk.ErrUnexpectedSpdkCallResult
	}

	delete(s.volumes.qosVolumes, in.Name)
	return &emptypb.Empty{}, nil
}

func (s *Server) verifyQosVolume(volume *pb.QosVolume) error {
	if volume.QosVolumeId == nil || volume.QosVolumeId.Value == "" {
		return fmt.Errorf("qos_volume_id cannot be empty")
	}
	if volume.VolumeId == nil || volume.VolumeId.Value == "" {
		return fmt.Errorf("volume_id cannot be empty")
	}

	if volume.LimitMin != nil {
		return fmt.Errorf("QoS volume limit_min is not supported")
	}
	if volume.LimitMax.RdIopsKiops != 0 {
		return fmt.Errorf("QoS volume limit_max rd_iops_kiops is not supported")
	}
	if volume.LimitMax.WrIopsKiops != 0 {
		return fmt.Errorf("QoS volume limit_max wr_iops_kiops is not supported")
	}

	if volume.LimitMax.RdBandwidthMbs == 0 &&
		volume.LimitMax.WrBandwidthMbs == 0 &&
		volume.LimitMax.RwBandwidthMbs == 0 &&
		volume.LimitMax.RdIopsKiops == 0 &&
		volume.LimitMax.WrIopsKiops == 0 &&
		volume.LimitMax.RwIopsKiops == 0 {
		return fmt.Errorf("QoS volume limit_max should set limit")
	}

	if volume.LimitMax.RwIopsKiops < 0 {
		return fmt.Errorf("QoS volume limit_max rw_iops_kiops cannot be negative")
	}
	if volume.LimitMax.RdBandwidthMbs < 0 {
		return fmt.Errorf("QoS volume limit_max rd_bandwidth_mbs cannot be negative")
	}
	if volume.LimitMax.WrBandwidthMbs < 0 {
		return fmt.Errorf("QoS volume limit_max wr_bandwidth_mbs cannot be negative")
	}
	if volume.LimitMax.RwBandwidthMbs < 0 {
		return fmt.Errorf("QoS volume limit_max rw_bandwidth_mbs cannot be negative")
	}

	return nil
}