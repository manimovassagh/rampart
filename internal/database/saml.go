package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// CreateSAMLProvider inserts a new SAML provider configuration.
func (db *DB) CreateSAMLProvider(ctx context.Context, p *model.SAMLProvider) (*model.SAMLProvider, error) {
	attrJSON, _ := json.Marshal(p.AttributeMapping)

	var out model.SAMLProvider
	var attrBytes []byte
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO saml_providers (org_id, name, entity_id, metadata_url, metadata_xml, sso_url, slo_url, certificate, name_id_format, attribute_mapping, enabled)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, org_id, name, entity_id, metadata_url, metadata_xml, sso_url, slo_url, certificate, name_id_format, attribute_mapping, enabled, created_at, updated_at`,
		p.OrgID, p.Name, p.EntityID, p.MetadataURL, p.MetadataXML,
		p.SSOURL, p.SLOURL, p.Certificate, p.NameIDFormat, attrJSON, p.Enabled,
	).Scan(&out.ID, &out.OrgID, &out.Name, &out.EntityID, &out.MetadataURL, &out.MetadataXML,
		&out.SSOURL, &out.SLOURL, &out.Certificate, &out.NameIDFormat,
		&attrBytes, &out.Enabled, &out.CreatedAt, &out.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating SAML provider: %w", err)
	}
	_ = json.Unmarshal(attrBytes, &out.AttributeMapping)
	return &out, nil
}

// GetSAMLProviderByID returns a SAML provider by ID.
func (db *DB) GetSAMLProviderByID(ctx context.Context, id uuid.UUID) (*model.SAMLProvider, error) {
	var p model.SAMLProvider
	var attrBytes []byte
	err := db.Pool.QueryRow(ctx,
		`SELECT id, org_id, name, entity_id, metadata_url, metadata_xml, sso_url, slo_url, certificate, name_id_format, attribute_mapping, enabled, created_at, updated_at
		 FROM saml_providers WHERE id = $1`, id,
	).Scan(&p.ID, &p.OrgID, &p.Name, &p.EntityID, &p.MetadataURL, &p.MetadataXML,
		&p.SSOURL, &p.SLOURL, &p.Certificate, &p.NameIDFormat,
		&attrBytes, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting SAML provider: %w", err)
	}
	_ = json.Unmarshal(attrBytes, &p.AttributeMapping)
	return &p, nil
}

// ListSAMLProviders returns all SAML providers for an organization.
func (db *DB) ListSAMLProviders(ctx context.Context, orgID uuid.UUID) ([]*model.SAMLProvider, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, org_id, name, entity_id, metadata_url, metadata_xml, sso_url, slo_url, certificate, name_id_format, attribute_mapping, enabled, created_at, updated_at
		 FROM saml_providers WHERE org_id = $1 ORDER BY name ASC`, orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing SAML providers: %w", err)
	}
	defer rows.Close()

	var providers []*model.SAMLProvider
	for rows.Next() {
		var p model.SAMLProvider
		var attrBytes []byte
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.EntityID, &p.MetadataURL, &p.MetadataXML,
			&p.SSOURL, &p.SLOURL, &p.Certificate, &p.NameIDFormat,
			&attrBytes, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning SAML provider: %w", err)
		}
		_ = json.Unmarshal(attrBytes, &p.AttributeMapping)
		providers = append(providers, &p)
	}
	return providers, nil
}

// GetEnabledSAMLProviders returns enabled SAML providers for an organization.
func (db *DB) GetEnabledSAMLProviders(ctx context.Context, orgID uuid.UUID) ([]*model.SAMLProvider, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, org_id, name, entity_id, metadata_url, metadata_xml, sso_url, slo_url, certificate, name_id_format, attribute_mapping, enabled, created_at, updated_at
		 FROM saml_providers WHERE org_id = $1 AND enabled = true ORDER BY name ASC`, orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("getting enabled SAML providers: %w", err)
	}
	defer rows.Close()

	var providers []*model.SAMLProvider
	for rows.Next() {
		var p model.SAMLProvider
		var attrBytes []byte
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.EntityID, &p.MetadataURL, &p.MetadataXML,
			&p.SSOURL, &p.SLOURL, &p.Certificate, &p.NameIDFormat,
			&attrBytes, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning SAML provider: %w", err)
		}
		_ = json.Unmarshal(attrBytes, &p.AttributeMapping)
		providers = append(providers, &p)
	}
	return providers, nil
}

// UpdateSAMLProvider updates a SAML provider configuration.
func (db *DB) UpdateSAMLProvider(ctx context.Context, id uuid.UUID, req *model.UpdateSAMLProviderRequest) (*model.SAMLProvider, error) {
	attrJSON, _ := json.Marshal(req.AttributeMapping)

	var p model.SAMLProvider
	var attrBytes []byte
	err := db.Pool.QueryRow(ctx,
		`UPDATE saml_providers SET name = $2, entity_id = $3, metadata_url = $4, metadata_xml = $5,
		 sso_url = $6, slo_url = $7, certificate = $8, name_id_format = $9,
		 attribute_mapping = $10, enabled = $11, updated_at = now()
		 WHERE id = $1
		 RETURNING id, org_id, name, entity_id, metadata_url, metadata_xml, sso_url, slo_url, certificate, name_id_format, attribute_mapping, enabled, created_at, updated_at`,
		id, req.Name, req.EntityID, req.MetadataURL, req.MetadataXML,
		req.SSOURL, req.SLOURL, req.Certificate, req.NameIDFormat, attrJSON, req.Enabled,
	).Scan(&p.ID, &p.OrgID, &p.Name, &p.EntityID, &p.MetadataURL, &p.MetadataXML,
		&p.SSOURL, &p.SLOURL, &p.Certificate, &p.NameIDFormat,
		&attrBytes, &p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("updating SAML provider: %w", err)
	}
	_ = json.Unmarshal(attrBytes, &p.AttributeMapping)
	return &p, nil
}

// DeleteSAMLProvider deletes a SAML provider.
func (db *DB) DeleteSAMLProvider(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM saml_providers WHERE id = $1`, id)
	return err
}
