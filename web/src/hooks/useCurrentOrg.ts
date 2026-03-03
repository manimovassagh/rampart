import { useState, useEffect, useCallback } from "react";
import { getMe } from "../api/auth";
import { getOrg } from "../api/organizations";
import type { OrgResponse } from "../types";

const ACTIVE_ORG_KEY = "rampart_active_org_id";

let cachedOrg: OrgResponse | null = null;
let defaultOrgId: string | null = null;
let fetchPromise: Promise<OrgResponse | null> | null = null;
const listeners = new Set<(org: OrgResponse | null) => void>();

function notify(org: OrgResponse | null) {
  listeners.forEach((fn) => fn(org));
}

async function resolveOrg(): Promise<OrgResponse | null> {
  const me = await getMe();
  if (!me) return null;
  defaultOrgId = me.org_id;

  // Check if user previously selected a different org
  const savedOrgId = localStorage.getItem(ACTIVE_ORG_KEY);
  const targetOrgId = savedOrgId || me.org_id;

  return getOrg(targetOrgId);
}

function fetchOnce(): Promise<OrgResponse | null> {
  if (!fetchPromise) {
    fetchPromise = resolveOrg().catch(() => null);
  }
  return fetchPromise;
}

export function getActiveOrgId(): string | null {
  return cachedOrg?.id ?? localStorage.getItem(ACTIVE_ORG_KEY) ?? defaultOrgId;
}

export function clearOrgCache() {
  cachedOrg = null;
  defaultOrgId = null;
  fetchPromise = null;
  localStorage.removeItem(ACTIVE_ORG_KEY);
}

export function useCurrentOrg() {
  const [org, setOrg] = useState<OrgResponse | null>(cachedOrg);
  const [loading, setLoading] = useState(!cachedOrg);

  useEffect(() => {
    // Subscribe to org changes from other components
    const handler = (updated: OrgResponse | null) => setOrg(updated);
    listeners.add(handler);

    if (!cachedOrg) {
      fetchOnce().then((result) => {
        cachedOrg = result;
        setOrg(result);
        setLoading(false);
      });
    }

    return () => {
      listeners.delete(handler);
    };
  }, []);

  const switchOrg = useCallback(async (orgId: string) => {
    try {
      const newOrg = await getOrg(orgId);
      cachedOrg = newOrg;
      localStorage.setItem(ACTIVE_ORG_KEY, orgId);
      // Reset fetch promise so future calls use the new org
      fetchPromise = null;
      setOrg(newOrg);
      notify(newOrg);
    } catch {
      // If org fetch fails, stay on current org
    }
  }, []);

  return { org, loading, switchOrg, defaultOrgId };
}
