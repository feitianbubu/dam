package example

import (
	"errors"
	"fmt"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	exampleReq "github.com/flipped-aurora/gin-vue-admin/server/model/example/request"
	"github.com/flipped-aurora/gin-vue-admin/server/utils"
)

type ProjectService struct{}

// CreateProject 创建项目
func (s *ProjectService) CreateProject(req exampleReq.ProjectCreateRequest, userID uint) (example.Project, error) {
	project := example.Project{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
		IsPublic:    req.IsPublic,
	}

	if err := global.GVA_DB.Create(&project).Error; err != nil {
		return project, err
	}

	return project, nil
}

// UpdateProject 更新项目
func (s *ProjectService) UpdateProject(req exampleReq.ProjectUpdateRequest, userID uint) error {
	var project example.Project
	if err := global.GVA_DB.First(&project, req.ID).Error; err != nil {
		return errors.New("项目不存在")
	}

	// 只有项目所有者可以更新项目
	if project.OwnerID != userID {
		return errors.New("只有项目所有者可以更新项目")
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}

	if len(updates) > 0 {
		if err := global.GVA_DB.Model(&project).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

// DeleteProject 删除项目
func (s *ProjectService) DeleteProject(id uint, userID uint) error {
	var project example.Project
	if err := global.GVA_DB.First(&project, id).Error; err != nil {
		return errors.New("项目不存在")
	}

	// 只有项目所有者可以删除项目
	if project.OwnerID != userID {
		return errors.New("只有项目所有者可以删除项目")
	}

	return global.GVA_DB.Delete(&project).Error
}

// GetProjectInfo 获取项目详情
func (s *ProjectService) GetProjectInfo(id uint) (example.Project, error) {
	var project example.Project
	err := global.GVA_DB.First(&project, id).Error
	return project, err
}

// GetProjectList 获取项目列表
func (s *ProjectService) GetProjectList(req exampleReq.ProjectListRequest, userID uint) (list []example.Project, total int64, err error) {
	db := global.GVA_DB.Model(&example.Project{})

	// 按名称模糊搜索
	if req.Name != "" {
		db = db.Where("name LIKE ?", "%"+req.Name+"%")
	}

	// 按可见性筛选
	if req.IsPublic != nil {
		db = db.Where("is_public = ?", *req.IsPublic)
	}

	// 只显示用户拥有的项目或公开的项目
	db = db.Where("owner_id = ? OR is_public = ?", userID, true)

	err = db.Count(&total).Error
	if err != nil {
		return
	}

	limit := req.PageSize
	offset := req.PageSize * (req.Page - 1)
	err = db.Limit(limit).Offset(offset).Order("created_at DESC").Find(&list).Error

	return list, total, err
}

// SetProjectPermission 设置项目权限（Casbin策略）
func (s *ProjectService) SetProjectPermission(req exampleReq.ProjectPermissionRequest) error {
	// 验证项目是否存在
	var project example.Project
	if err := global.GVA_DB.First(&project, "id = ?", req.ProjectID).Error; err != nil {
		return errors.New("项目不存在")
	}

	// 构建 Casbin 策略
	sub := fmt.Sprintf("%d", req.AuthorityID)
	obj := fmt.Sprintf("/project/%s", req.ProjectID)
	act := req.Permission // read 或 write

	// 添加策略到 Casbin
	e := utils.GetCasbin()
	success, err := e.AddPolicy(sub, obj, act)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("策略已存在")
	}

	return nil
}

// RemoveProjectPermission 移除项目权限
func (s *ProjectService) RemoveProjectPermission(req exampleReq.ProjectPermissionRequest) error {
	sub := fmt.Sprintf("%d", req.AuthorityID)
	obj := fmt.Sprintf("/project/%s", req.ProjectID)
	act := req.Permission

	e := utils.GetCasbin()
	success, err := e.RemovePolicy(sub, obj, act)
	if err != nil {
		return err
	}

	if !success {
		return errors.New("策略不存在")
	}

	return nil
}

// GetProjectPermissions 获取项目的所有权限策略
func (s *ProjectService) GetProjectPermissions(projectID string) ([][]string, error) {
	// 验证项目是否存在
	var project example.Project
	if err := global.GVA_DB.First(&project, "id = ?", projectID).Error; err != nil {
		return nil, errors.New("项目不存在")
	}

	e := utils.GetCasbin()
	obj := fmt.Sprintf("/project/%s", projectID)

	// 获取所有相关的策略
	policies, _ := e.GetFilteredPolicy(1, obj)

	return policies, nil
}
