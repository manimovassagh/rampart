export interface RegistrationRequest {
  username: string;
  email: string;
  password: string;
  given_name: string;
  family_name: string;
}

export interface UserResponse {
  id: string;
  org_id: string;
  username: string;
  email: string;
  email_verified: boolean;
  given_name: string;
  family_name: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface FieldError {
  field: string;
  message: string;
}

export interface ValidationErrorResponse {
  error: string;
  error_description: string;
  status: number;
  request_id?: string;
  fields: FieldError[];
}

export interface ApiErrorResponse {
  error: string;
  error_description: string;
  status: number;
  request_id?: string;
}

export interface LoginRequest {
  identifier: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
  user: UserResponse;
}

export interface RefreshResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

export interface MeResponse {
  id: string;
  org_id: string;
  preferred_username: string;
  email: string;
  email_verified: boolean;
  given_name?: string;
  family_name?: string;
}

// Admin types

export interface AdminUserResponse {
  id: string;
  org_id: string;
  username: string;
  email: string;
  email_verified: boolean;
  given_name?: string;
  family_name?: string;
  enabled: boolean;
  mfa_enabled: boolean;
  last_login_at?: string;
  session_count: number;
  created_at: string;
  updated_at: string;
}

export interface ListUsersResponse {
  users: AdminUserResponse[];
  total: number;
  page: number;
  limit: number;
}

export interface CreateUserRequest {
  username: string;
  email: string;
  password: string;
  given_name: string;
  family_name: string;
  enabled: boolean;
  email_verified: boolean;
}

export interface UpdateUserRequest {
  username: string;
  email: string;
  given_name: string;
  family_name: string;
  enabled: boolean;
  email_verified: boolean;
}

export interface ResetPasswordRequest {
  password: string;
}

export interface DashboardStats {
  total_users: number;
  active_sessions: number;
  recent_users: number;
  total_organizations: number;
}

// Organization types

export interface OrgResponse {
  id: string;
  name: string;
  slug: string;
  display_name: string;
  enabled: boolean;
  user_count: number;
  created_at: string;
  updated_at: string;
}

export interface ListOrgsResponse {
  organizations: OrgResponse[];
  total: number;
  page: number;
  limit: number;
}

export interface CreateOrgRequest {
  name: string;
  slug: string;
  display_name: string;
}

export interface UpdateOrgRequest {
  name: string;
  display_name: string;
  enabled: boolean;
}

export interface OrgSettingsResponse {
  id: string;
  org_id: string;
  password_min_length: number;
  password_require_uppercase: boolean;
  password_require_lowercase: boolean;
  password_require_numbers: boolean;
  password_require_symbols: boolean;
  mfa_enforcement: string;
  access_token_ttl_seconds: number;
  refresh_token_ttl_seconds: number;
  logo_url?: string;
  primary_color?: string;
  background_color?: string;
  created_at: string;
  updated_at: string;
}

export interface UpdateOrgSettingsRequest {
  password_min_length: number;
  password_require_uppercase: boolean;
  password_require_lowercase: boolean;
  password_require_numbers: boolean;
  password_require_symbols: boolean;
  mfa_enforcement: string;
  access_token_ttl_seconds: number;
  refresh_token_ttl_seconds: number;
  logo_url: string;
  primary_color: string;
  background_color: string;
}

export interface SessionResponse {
  id: string;
  created_at: string;
  expires_at: string;
}
