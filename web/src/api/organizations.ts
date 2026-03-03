import { authFetch, authJSON } from "./admin";
import type {
  ListOrgsResponse,
  OrgResponse,
  CreateOrgRequest,
  UpdateOrgRequest,
  OrgSettingsResponse,
  UpdateOrgSettingsRequest,
} from "../types";

export async function listOrgs(
  page = 1,
  limit = 20,
  search = "",
): Promise<ListOrgsResponse> {
  const params = new URLSearchParams({
    page: String(page),
    limit: String(limit),
  });
  if (search) params.set("search", search);
  return authJSON<ListOrgsResponse>(
    `/api/v1/admin/organizations?${params}`,
  );
}

export async function getOrg(id: string): Promise<OrgResponse> {
  return authJSON<OrgResponse>(`/api/v1/admin/organizations/${id}`);
}

export async function createOrg(
  data: CreateOrgRequest,
): Promise<OrgResponse> {
  return authJSON<OrgResponse>("/api/v1/admin/organizations", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function updateOrg(
  id: string,
  data: UpdateOrgRequest,
): Promise<OrgResponse> {
  return authJSON<OrgResponse>(`/api/v1/admin/organizations/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export async function deleteOrg(id: string): Promise<void> {
  const res = await authFetch(`/api/v1/admin/organizations/${id}`, {
    method: "DELETE",
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(
      (body as { error_description?: string }).error_description ??
        `Delete failed: ${res.status}`,
    );
  }
}

export async function getOrgSettings(
  id: string,
): Promise<OrgSettingsResponse> {
  return authJSON<OrgSettingsResponse>(
    `/api/v1/admin/organizations/${id}/settings`,
  );
}

export async function updateOrgSettings(
  id: string,
  data: UpdateOrgSettingsRequest,
): Promise<OrgSettingsResponse> {
  return authJSON<OrgSettingsResponse>(
    `/api/v1/admin/organizations/${id}/settings`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    },
  );
}
