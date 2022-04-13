//go:build linux && cgo && !agent

package db

// The code below was generated by lxd-generate - DO NOT EDIT!

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/lxc/lxd/lxd/db/cluster"
	"github.com/lxc/lxd/lxd/db/query"
	"github.com/lxc/lxd/shared/api"
)

var _ = api.ServerEnvironment{}

var projectObjects = cluster.RegisterStmt(`
SELECT projects.id, projects.description, projects.name
  FROM projects
  ORDER BY projects.name
`)

var projectObjectsByName = cluster.RegisterStmt(`
SELECT projects.id, projects.description, projects.name
  FROM projects
  WHERE projects.name = ? ORDER BY projects.name
`)

var projectObjectsByID = cluster.RegisterStmt(`
SELECT projects.id, projects.description, projects.name
  FROM projects
  WHERE projects.id = ? ORDER BY projects.name
`)

var projectCreate = cluster.RegisterStmt(`
INSERT INTO projects (description, name)
  VALUES (?, ?)
`)

var projectID = cluster.RegisterStmt(`
SELECT projects.id FROM projects
  WHERE projects.name = ?
`)

var projectRename = cluster.RegisterStmt(`
UPDATE projects SET name = ? WHERE name = ?
`)

var projectUpdate = cluster.RegisterStmt(`
UPDATE projects
  SET description = ?
 WHERE id = ?
`)

var projectDeleteByName = cluster.RegisterStmt(`
DELETE FROM projects WHERE name = ?
`)

// GetProjects returns all available projects.
// generator: project GetMany
func (c *ClusterTx) GetProjects(filter ProjectFilter) ([]Project, error) {
	var err error

	// Result slice.
	objects := make([]Project, 0)

	// Pick the prepared statement and arguments to use based on active criteria.
	var stmt *sql.Stmt
	var args []any

	if filter.Name != nil && filter.ID == nil {
		stmt = c.stmt(projectObjectsByName)
		args = []any{
			filter.Name,
		}
	} else if filter.ID != nil && filter.Name == nil {
		stmt = c.stmt(projectObjectsByID)
		args = []any{
			filter.ID,
		}
	} else if filter.ID == nil && filter.Name == nil {
		stmt = c.stmt(projectObjects)
		args = []any{}
	} else {
		return nil, fmt.Errorf("No statement exists for the given Filter")
	}

	// Dest function for scanning a row.
	dest := func(i int) []any {
		objects = append(objects, Project{})
		return []any{
			&objects[i].ID,
			&objects[i].Description,
			&objects[i].Name,
		}
	}

	// Select.
	err = query.SelectObjects(stmt, dest, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"projects\" table: %w", err)
	}

	return objects, nil
}

// GetProjectConfig returns all available Project Config
// generator: project GetMany
func (c *ClusterTx) GetProjectConfig(projectID int) (map[string]string, error) {
	projectConfig, err := c.GetConfig("project")
	if err != nil {
		return nil, err
	}

	config, ok := projectConfig[projectID]
	if !ok {
		config = map[string]string{}
	}
	return config, nil
}

// GetProject returns the project with the given key.
// generator: project GetOne
func (c *ClusterTx) GetProject(name string) (*Project, error) {
	filter := ProjectFilter{}
	filter.Name = &name

	objects, err := c.GetProjects(filter)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"projects\" table: %w", err)
	}

	switch len(objects) {
	case 0:
		return nil, api.StatusErrorf(http.StatusNotFound, "Project not found")
	case 1:
		return &objects[0], nil
	default:
		return nil, fmt.Errorf("More than one \"projects\" entry matches")
	}
}

// ProjectExists checks if a project with the given key exists.
// generator: project Exists
func (c *ClusterTx) ProjectExists(name string) (bool, error) {
	_, err := c.GetProjectID(name)
	if err != nil {
		if api.StatusErrorCheck(err, http.StatusNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CreateProject adds a new project to the database.
// generator: project Create
func (c *ClusterTx) CreateProject(object Project) (int64, error) {
	// Check if a project with the same key exists.
	exists, err := c.ProjectExists(object.Name)
	if err != nil {
		return -1, fmt.Errorf("Failed to check for duplicates: %w", err)
	}

	if exists {
		return -1, fmt.Errorf("This \"projects\" entry already exists")
	}

	args := make([]any, 2)

	// Populate the statement arguments.
	args[0] = object.Description
	args[1] = object.Name

	// Prepared statement to use.
	stmt := c.stmt(projectCreate)

	// Execute the statement.
	result, err := stmt.Exec(args...)
	if err != nil {
		return -1, fmt.Errorf("Failed to create \"projects\" entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return -1, fmt.Errorf("Failed to fetch \"projects\" entry ID: %w", err)
	}

	return id, nil
}

// CreateProjectConfig adds a new project Config to the database.
// generator: project Create
func (c *ClusterTx) CreateProjectConfig(projectID int64, config map[string]string) error {
	referenceID := int(projectID)
	for key, value := range config {
		insert := Config{
			ReferenceID: referenceID,
			Key:         key,
			Value:       value,
		}

		err := c.CreateConfig("project", insert)
		if err != nil {
			return fmt.Errorf("Insert Config failed for Project: %w", err)
		}

	}
	return nil
}

// GetProjectID return the ID of the project with the given key.
// generator: project ID
func (c *ClusterTx) GetProjectID(name string) (int64, error) {
	stmt := c.stmt(projectID)
	rows, err := stmt.Query(name)
	if err != nil {
		return -1, fmt.Errorf("Failed to get \"projects\" ID: %w", err)
	}

	defer rows.Close()

	// Ensure we read one and only one row.
	if !rows.Next() {
		return -1, api.StatusErrorf(http.StatusNotFound, "Project not found")
	}
	var id int64
	err = rows.Scan(&id)
	if err != nil {
		return -1, fmt.Errorf("Failed to scan ID: %w", err)
	}

	if rows.Next() {
		return -1, fmt.Errorf("More than one row returned")
	}
	err = rows.Err()
	if err != nil {
		return -1, fmt.Errorf("Result set failure: %w", err)
	}

	return id, nil
}

// RenameProject renames the project matching the given key parameters.
// generator: project Rename
func (c *ClusterTx) RenameProject(name string, to string) error {
	stmt := c.stmt(projectRename)
	result, err := stmt.Exec(to, name)
	if err != nil {
		return fmt.Errorf("Rename Project failed: %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows failed: %w", err)
	}

	if n != 1 {
		return fmt.Errorf("Query affected %d rows instead of 1", n)
	}
	return nil
}

// DeleteProject deletes the project matching the given key parameters.
// generator: project DeleteOne-by-Name
func (c *ClusterTx) DeleteProject(name string) error {
	stmt := c.stmt(projectDeleteByName)
	result, err := stmt.Exec(name)
	if err != nil {
		return fmt.Errorf("Delete \"projects\": %w", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	if n != 1 {
		return fmt.Errorf("Query deleted %d rows instead of 1", n)
	}

	return nil
}
