package model

import (
	"time"

	"github.com/google/uuid"
)

// SAMLProvider represents a configured SAML Identity Provider for SP-initiated SSO.
type SAMLProvider struct {
	ID               uuid.UUID
	OrgID            uuid.UUID
	Name             string
	EntityID         string
	MetadataURL      string
	MetadataXML      string
	SSOURL           string
	SLOURL           string
	Certificate      string
	NameIDFormat     string
	AttributeMapping map[string]string // maps SAML attributes to Rampart fields (email, given_name, family_name, username)
	Enabled          bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CreateSAMLProviderRequest is the input for creating a SAML provider.
type CreateSAMLProviderRequest struct {
	Name             string            `json:"name"`
	EntityID         string            `json:"entity_id"`
	MetadataURL      string            `json:"metadata_url"`
	MetadataXML      string            `json:"metadata_xml"`
	SSOURL           string            `json:"sso_url"`
	SLOURL           string            `json:"slo_url"`
	Certificate      string            `json:"certificate"`
	NameIDFormat     string            `json:"name_id_format"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
}

// UpdateSAMLProviderRequest is the input for updating a SAML provider.
type UpdateSAMLProviderRequest struct {
	Name             string            `json:"name"`
	EntityID         string            `json:"entity_id"`
	MetadataURL      string            `json:"metadata_url"`
	MetadataXML      string            `json:"metadata_xml"`
	SSOURL           string            `json:"sso_url"`
	SLOURL           string            `json:"slo_url"`
	Certificate      string            `json:"certificate"`
	NameIDFormat     string            `json:"name_id_format"`
	AttributeMapping map[string]string `json:"attribute_mapping"`
	Enabled          bool              `json:"enabled"`
}
