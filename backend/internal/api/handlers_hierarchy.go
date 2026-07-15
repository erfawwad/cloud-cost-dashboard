package api

import (
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"

	appauth "cloudcostdash/internal/auth"
	"cloudcostdash/internal/models"
)

// ---- create/delete handlers ------------------------------------------------

func (s *Server) handleCreateOrganization(w http.ResponseWriter, r *http.Request) {
	var body struct{ Name string `json:"name"` }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	org := models.Organization{Name: body.Name}
	if err := s.DB.Create(&org).Error; err != nil {
		writeError(w, http.StatusConflict, "could not create organization (name may already exist)")
		return
	}
	writeJSON(w, http.StatusCreated, org)
}

func (s *Server) handleDeleteOrganization(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		var products []models.Product
		if err := tx.Where("organization_id = ?", id).Find(&products).Error; err != nil {
			return err
		}
		for _, p := range products {
			if err := cascadeDeleteProduct(tx, p.ID); err != nil {
				return err
			}
		}
		return tx.Delete(&models.Organization{}, id).Error
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete organization")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OrganizationID uint   `json:"organizationId"`
		Name           string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.OrganizationID == 0 {
		writeError(w, http.StatusBadRequest, "organizationId and name are required")
		return
	}
	product := models.Product{OrganizationID: body.OrganizationID, Name: body.Name}
	if err := s.DB.Create(&product).Error; err != nil {
		writeError(w, http.StatusBadRequest, "could not create product")
		return
	}
	writeJSON(w, http.StatusCreated, product)
}

func (s *Server) handleDeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		return cascadeDeleteProduct(tx, id)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete product")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProductID uint   `json:"productId"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.ProductID == 0 {
		writeError(w, http.StatusBadRequest, "productId and name are required")
		return
	}
	project := models.Project{ProductID: body.ProductID, Name: body.Name}
	if err := s.DB.Create(&project).Error; err != nil {
		writeError(w, http.StatusBadRequest, "could not create project")
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		return cascadeDeleteProject(tx, id)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateEnvironment(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProjectID uint   `json:"projectId"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.ProjectID == 0 {
		writeError(w, http.StatusBadRequest, "projectId and name are required")
		return
	}
	env := models.Environment{ProjectID: body.ProjectID, Name: body.Name}
	if err := s.DB.Create(&env).Error; err != nil {
		writeError(w, http.StatusBadRequest, "could not create environment")
		return
	}
	writeJSON(w, http.StatusCreated, env)
}

func (s *Server) handleDeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		return cascadeDeleteEnvironment(tx, id)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete environment")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleCreateRegion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EnvironmentID uint   `json:"environmentId"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.EnvironmentID == 0 {
		writeError(w, http.StatusBadRequest, "environmentId and name are required")
		return
	}
	region := models.Region{EnvironmentID: body.EnvironmentID, Name: body.Name}
	if err := s.DB.Create(&region).Error; err != nil {
		writeError(w, http.StatusBadRequest, "could not create region")
		return
	}
	writeJSON(w, http.StatusCreated, region)
}

func (s *Server) handleDeleteRegion(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		return cascadeDeleteRegion(tx, id)
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete region")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- cascade delete helpers -------------------------------------------------
// Deleting a hierarchy node also deletes everything beneath it (cloud
// accounts and their cost history included) so nothing is silently orphaned.

func cascadeDeleteRegion(tx *gorm.DB, regionID uint) error {
	var accounts []models.CloudAccount
	if err := tx.Where("region_id = ?", regionID).Find(&accounts).Error; err != nil {
		return err
	}
	for _, a := range accounts {
		if err := tx.Where("cloud_account_id = ?", a.ID).Delete(&models.CostRecord{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("region_id = ?", regionID).Delete(&models.CloudAccount{}).Error; err != nil {
		return err
	}
	return tx.Delete(&models.Region{}, regionID).Error
}

func cascadeDeleteEnvironment(tx *gorm.DB, envID uint) error {
	var regions []models.Region
	if err := tx.Where("environment_id = ?", envID).Find(&regions).Error; err != nil {
		return err
	}
	for _, reg := range regions {
		if err := cascadeDeleteRegion(tx, reg.ID); err != nil {
			return err
		}
	}
	return tx.Delete(&models.Environment{}, envID).Error
}

func cascadeDeleteProject(tx *gorm.DB, projectID uint) error {
	var envs []models.Environment
	if err := tx.Where("project_id = ?", projectID).Find(&envs).Error; err != nil {
		return err
	}
	for _, e := range envs {
		if err := cascadeDeleteEnvironment(tx, e.ID); err != nil {
			return err
		}
	}
	return tx.Delete(&models.Project{}, projectID).Error
}

func cascadeDeleteProduct(tx *gorm.DB, productID uint) error {
	var projects []models.Project
	if err := tx.Where("product_id = ?", productID).Find(&projects).Error; err != nil {
		return err
	}
	for _, p := range projects {
		if err := cascadeDeleteProject(tx, p.ID); err != nil {
			return err
		}
	}
	return tx.Delete(&models.Product{}, productID).Error
}

// ---- tree (hierarchy + cost rollup) ----------------------------------------

type cloudAccountTree struct {
	ID          uint                `json:"id"`
	Name        string              `json:"name"`
	Provider    models.ProviderKey  `json:"provider"`
	ExternalID  string              `json:"externalId"`
	Active      bool                `json:"active"`
	LastSyncAt  *time.Time          `json:"lastSyncAt,omitempty"`
	LastSyncErr string              `json:"lastSyncErr,omitempty"`
	TotalCost   float64             `json:"totalCost"`
}

type regionTree struct {
	ID            uint               `json:"id"`
	Name          string             `json:"name"`
	TotalCost     float64            `json:"totalCost"`
	CloudAccounts []cloudAccountTree `json:"cloudAccounts"`
}

type environmentTree struct {
	ID        uint         `json:"id"`
	Name      string       `json:"name"`
	TotalCost float64      `json:"totalCost"`
	Regions   []regionTree `json:"regions"`
}

type projectTree struct {
	ID           uint              `json:"id"`
	Name         string            `json:"name"`
	TotalCost    float64           `json:"totalCost"`
	Environments []environmentTree `json:"environments"`
}

type productTree struct {
	ID        uint          `json:"id"`
	Name      string        `json:"name"`
	TotalCost float64       `json:"totalCost"`
	Projects  []projectTree `json:"projects"`
}

type organizationTree struct {
	ID        uint          `json:"id"`
	Name      string        `json:"name"`
	TotalCost float64       `json:"totalCost"`
	Products  []productTree `json:"products"`
}

// handleTree returns the full org hierarchy with each node's total cost
// rolled up from its cloud accounts over the requested date range (default:
// last 30 days). Viewers only see the subtree their account is scoped to.
func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseDateRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var orgs []models.Organization
	query := s.DB.Preload("Products.Projects.Environments.Regions.CloudAccounts")
	if err := query.Find(&orgs).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load hierarchy")
		return
	}

	costByAccount, err := s.costByAccountID(start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load costs")
		return
	}

	claims, _ := appauth.FromContext(r.Context())

	result := make([]organizationTree, 0, len(orgs))
	for _, org := range orgs {
		orgNode := organizationTree{ID: org.ID, Name: org.Name, Products: []productTree{}}
		for _, product := range org.Products {
			if !productInScope(claims, product) {
				continue
			}
			productNode := productTree{ID: product.ID, Name: product.Name, Projects: []projectTree{}}
			for _, project := range product.Projects {
				if !projectInScope(claims, project) {
					continue
				}
				projectNode := projectTree{ID: project.ID, Name: project.Name, Environments: []environmentTree{}}
				for _, env := range project.Environments {
					envNode := environmentTree{ID: env.ID, Name: env.Name, Regions: []regionTree{}}
					for _, region := range env.Regions {
						regionNode := regionTree{ID: region.ID, Name: region.Name, CloudAccounts: []cloudAccountTree{}}
						for _, acct := range region.CloudAccounts {
							cost := costByAccount[acct.ID]
							regionNode.CloudAccounts = append(regionNode.CloudAccounts, cloudAccountTree{
								ID: acct.ID, Name: acct.Name, Provider: acct.Provider,
								ExternalID: acct.ExternalID, Active: acct.Active,
								LastSyncAt: acct.LastSyncAt, LastSyncErr: acct.LastSyncErr,
								TotalCost: cost,
							})
							regionNode.TotalCost += cost
						}
						envNode.Regions = append(envNode.Regions, regionNode)
						envNode.TotalCost += regionNode.TotalCost
					}
					projectNode.Environments = append(projectNode.Environments, envNode)
					projectNode.TotalCost += envNode.TotalCost
				}
				productNode.Projects = append(productNode.Projects, projectNode)
				productNode.TotalCost += projectNode.TotalCost
			}
			orgNode.Products = append(orgNode.Products, productNode)
			orgNode.TotalCost += productNode.TotalCost
		}
		result = append(result, orgNode)
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) costByAccountID(start, end time.Time) (map[uint]float64, error) {
	var sums []struct {
		CloudAccountID uint
		Total          float64
	}
	if err := s.DB.Model(&models.CostRecord{}).
		Select("cloud_account_id, sum(amount) as total").
		Where("date >= ? AND date <= ?", start, end).
		Group("cloud_account_id").
		Scan(&sums).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]float64, len(sums))
	for _, row := range sums {
		out[row.CloudAccountID] = row.Total
	}
	return out, nil
}

func productInScope(claims *appauth.Claims, product models.Product) bool {
	if claims == nil || claims.ScopeType == models.ScopeNone || claims.Role != models.RoleViewer {
		return true
	}
	if claims.ScopeType == models.ScopeProduct {
		return claims.ScopeID != nil && *claims.ScopeID == product.ID
	}
	// scoped to a project: keep the product, project-level filtering happens below
	return claims.ScopeType == models.ScopeProject
}

func projectInScope(claims *appauth.Claims, project models.Project) bool {
	if claims == nil || claims.ScopeType == models.ScopeNone || claims.Role != models.RoleViewer {
		return true
	}
	if claims.ScopeType == models.ScopeProject {
		return claims.ScopeID != nil && *claims.ScopeID == project.ID
	}
	return true // already filtered at the product level
}
