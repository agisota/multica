import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { runtimeKeys } from "./queries";
import type {
  CreateRuntimeLeaseRequest,
  UpdateRuntimeBillingRequest,
  UpdateRuntimePolicyRequest,
} from "../types";

export function useDeleteRuntime(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (runtimeId: string) => api.deleteRuntime(runtimeId),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: runtimeKeys.all(wsId) });
    },
  });
}

export function useCreateRuntimeLease(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateRuntimeLeaseRequest) => api.createRuntimeLease(data),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: runtimeKeys.leases(wsId) });
    },
  });
}

export function useStartRuntimeLease(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (leaseId: string) => api.startRuntimeLease(leaseId),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: runtimeKeys.leases(wsId) });
      qc.invalidateQueries({ queryKey: runtimeKeys.list(wsId) });
      qc.invalidateQueries({ queryKey: runtimeKeys.listMine(wsId) });
    },
  });
}

export function useStopRuntimeLease(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (leaseId: string) => api.stopRuntimeLease(leaseId),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: runtimeKeys.leases(wsId) });
      qc.invalidateQueries({ queryKey: runtimeKeys.list(wsId) });
      qc.invalidateQueries({ queryKey: runtimeKeys.listMine(wsId) });
    },
  });
}

export function useDeleteRuntimeLease(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (leaseId: string) => api.deleteRuntimeLease(leaseId),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: runtimeKeys.leases(wsId) });
      qc.invalidateQueries({ queryKey: runtimeKeys.list(wsId) });
      qc.invalidateQueries({ queryKey: runtimeKeys.listMine(wsId) });
    },
  });
}

export function useUpdateRuntimePolicy(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateRuntimePolicyRequest) => api.updateRuntimePolicy(data),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: runtimeKeys.policy(wsId) });
    },
  });
}

export function useUpdateRuntimeBilling(wsId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateRuntimeBillingRequest) => api.updateRuntimeBilling(data),
    onSettled: () => {
      qc.invalidateQueries({ queryKey: runtimeKeys.billing(wsId) });
    },
  });
}
