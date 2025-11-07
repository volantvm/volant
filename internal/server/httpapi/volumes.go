// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/volantvm/volant/internal/server/db"
	"github.com/volantvm/volant/internal/server/volumes"
)

type CreateVolumeRequest struct {
	Name       string `json:"name" binding:"required"`
	Type       string `json:"type" binding:"required"` // ext4|bind
	SizeGB     int    `json:"size_gb" binding:"required,min=1"`
	Persistent bool   `json:"persistent"`
	HostPath   string `json:"host_path"` // Required for bind mounts
}

type VolumeResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	SizeGB     int    `json:"size_gb"`
	Persistent bool   `json:"persistent"`
	HostPath   string `json:"host_path"`
	BackupPath string `json:"backup_path,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type VolumeMountResponse struct {
	VolumeName string `json:"volume_name"`
	MountPoint string `json:"mount_point"`
	ReadOnly   bool   `json:"read_only"`
}

func (api *apiServer) createVolume(c *gin.Context) {
	var req CreateVolumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate volume type
	if req.Type != volumes.VolumeTypeExt4 && req.Type != volumes.VolumeTypeBind {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid volume type: must be ext4 or bind"})
		return
	}

	// For bind mounts, host_path is required
	if req.Type == volumes.VolumeTypeBind && req.HostPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "host_path is required for bind volumes"})
		return
	}

	vol := &db.Volume{
		Name:       req.Name,
		Type:       req.Type,
		SizeGB:     req.SizeGB,
		Persistent: req.Persistent,
		HostPath:   req.HostPath,
	}

	// Get volume manager from engine's store
	volumeMgr := volumes.NewManager("/var/lib/volant/volumes", api.engine.Store())

	if err := volumeMgr.CreateVolume(c.Request.Context(), vol); err != nil {
		api.logger.Error("create volume", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	api.logger.Info("volume created", "name", vol.Name, "type", vol.Type, "size_gb", vol.SizeGB)

	c.JSON(http.StatusCreated, VolumeResponse{
		ID:         vol.ID,
		Name:       vol.Name,
		Type:       vol.Type,
		SizeGB:     vol.SizeGB,
		Persistent: vol.Persistent,
		HostPath:   vol.HostPath,
		BackupPath: vol.BackupPath,
		CreatedAt:  vol.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  vol.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (api *apiServer) listVolumes(c *gin.Context) {
	vols, err := api.engine.Store().Queries().Volumes().List(c.Request.Context())
	if err != nil {
		api.logger.Error("list volumes", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]VolumeResponse, 0, len(vols))
	for _, vol := range vols {
		response = append(response, VolumeResponse{
			ID:         vol.ID,
			Name:       vol.Name,
			Type:       vol.Type,
			SizeGB:     vol.SizeGB,
			Persistent: vol.Persistent,
			HostPath:   vol.HostPath,
			BackupPath: vol.BackupPath,
			CreatedAt:  vol.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:  vol.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, response)
}

func (api *apiServer) getVolume(c *gin.Context) {
	name := c.Param("name")
	vol, err := api.engine.Store().Queries().Volumes().GetByName(c.Request.Context(), name)
	if err != nil {
		api.logger.Error("get volume", "name", name, "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	// Get mounts for this volume
	mounts, _ := api.engine.Store().Queries().VolumeMounts().ListByVolume(c.Request.Context(), vol.ID)
	mountsResponse := make([]VolumeMountResponse, 0, len(mounts))
	for _, m := range mounts {
		mountsResponse = append(mountsResponse, VolumeMountResponse{
			VolumeName: vol.Name,
			MountPoint: m.MountPoint,
			ReadOnly:   m.ReadOnly,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"volume": VolumeResponse{
			ID:         vol.ID,
			Name:       vol.Name,
			Type:       vol.Type,
			SizeGB:     vol.SizeGB,
			Persistent: vol.Persistent,
			HostPath:   vol.HostPath,
			BackupPath: vol.BackupPath,
			CreatedAt:  vol.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:  vol.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		},
		"mounts": mountsResponse,
	})
}

func (api *apiServer) deleteVolume(c *gin.Context) {
	name := c.Param("name")
	volumeMgr := volumes.NewManager("/var/lib/volant/volumes", api.engine.Store())

	if err := volumeMgr.DeleteVolume(c.Request.Context(), name); err != nil {
		api.logger.Error("delete volume", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	api.logger.Info("volume deleted", "name", name)
	c.JSON(http.StatusOK, gin.H{"message": "volume deleted"})
}

func (api *apiServer) backupVolume(c *gin.Context) {
	name := c.Param("name")
	volumeMgr := volumes.NewManager("/var/lib/volant/volumes", api.engine.Store())

	if err := volumeMgr.BackupVolume(c.Request.Context(), name); err != nil {
		api.logger.Error("backup volume", "name", name, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	api.logger.Info("volume backed up", "name", name)
	c.JSON(http.StatusOK, gin.H{"message": "volume backed up"})
}
