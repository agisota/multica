"use client";

import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { toast } from "sonner";
import { runtimeBillingOptions, runtimeLeaseListOptions, runtimePolicyOptions } from "@multica/core/runtimes/queries";
import {
  useCreateRuntimeLease,
  useDeleteRuntimeLease,
  useStartRuntimeLease,
  useStopRuntimeLease,
  useUpdateRuntimeBilling,
  useUpdateRuntimePolicy,
} from "@multica/core/runtimes/mutations";
import type { RuntimeBackend, RuntimePlacement } from "@multica/core/types";
import { Button } from "@multica/ui/components/ui/button";
import { Input } from "@multica/ui/components/ui/input";
import { Skeleton } from "@multica/ui/components/ui/skeleton";
import { Switch } from "@multica/ui/components/ui/switch";

export function RuntimeOrchestrationPanel({ wsId }: { wsId: string }) {
  const { data: policy, isLoading: policyLoading } = useQuery(runtimePolicyOptions(wsId));
  const { data: billing, isLoading: billingLoading } = useQuery(runtimeBillingOptions(wsId));
  const { data: leases = [], isLoading: leasesLoading } = useQuery(runtimeLeaseListOptions(wsId));

  const updatePolicy = useUpdateRuntimePolicy(wsId);
  const updateBilling = useUpdateRuntimeBilling(wsId);
  const createLease = useCreateRuntimeLease(wsId);
  const startLease = useStartRuntimeLease(wsId);
  const stopLease = useStopRuntimeLease(wsId);
  const deleteLease = useDeleteRuntimeLease(wsId);

  const [sharedPoolEnabled, setSharedPoolEnabled] = useState(true);
  const [allowModal, setAllowModal] = useState(true);
  const [autoStopEnabled, setAutoStopEnabled] = useState(true);
  const [billingEnabled, setBillingEnabled] = useState(false);
  const [sharedHourlyRate, setSharedHourlyRate] = useState("0");
  const [privateHourlyRate, setPrivateHourlyRate] = useState("0");
  const [tokenMarkupPercent, setTokenMarkupPercent] = useState("0");
  const [leaseName, setLeaseName] = useState("");
  const [leaseBackend, setLeaseBackend] = useState<RuntimeBackend>("local");
  const [leasePlacement, setLeasePlacement] = useState<RuntimePlacement>("private");

  useEffect(() => {
    if (!policy) {
      return;
    }
    setSharedPoolEnabled(policy.shared_pool_enabled);
    setAllowModal(policy.allow_modal);
    setAutoStopEnabled(policy.auto_stop_enabled);
  }, [policy]);

  useEffect(() => {
    if (!billing) {
      return;
    }
    setBillingEnabled(billing.billing_enabled);
    setSharedHourlyRate(String(billing.shared_hourly_rate_cents));
    setPrivateHourlyRate(String(billing.private_hourly_rate_cents));
    setTokenMarkupPercent(String(billing.token_markup_percent));
  }, [billing]);

  const handleSavePolicy = () => {
    updatePolicy.mutate(
      {
        shared_pool_enabled: sharedPoolEnabled,
        allow_modal: allowModal,
        auto_stop_enabled: autoStopEnabled,
      },
      {
        onSuccess: () => toast.success("Runtime policy updated"),
        onError: (error) => toast.error(error instanceof Error ? error.message : "Failed to update runtime policy"),
      },
    );
  };

  const handleSaveBilling = () => {
    updateBilling.mutate(
      {
        billing_enabled: billingEnabled,
        shared_hourly_rate_cents: Number(sharedHourlyRate) || 0,
        private_hourly_rate_cents: Number(privateHourlyRate) || 0,
        token_markup_percent: Number(tokenMarkupPercent) || 0,
      },
      {
        onSuccess: () => toast.success("Runtime billing updated"),
        onError: (error) => toast.error(error instanceof Error ? error.message : "Failed to update runtime billing"),
      },
    );
  };

  const handleCreateLease = () => {
    if (!leaseName.trim()) {
      toast.error("Lease name is required");
      return;
    }

    createLease.mutate(
      {
        name: leaseName.trim(),
        backend: leaseBackend,
        placement: leasePlacement,
        scope: leasePlacement === "shared" ? "workspace" : "user",
      },
      {
        onSuccess: () => {
          toast.success("Runtime lease created");
          setLeaseName("");
        },
        onError: (error) => toast.error(error instanceof Error ? error.message : "Failed to create runtime lease"),
      },
    );
  };

  if (policyLoading || billingLoading || leasesLoading) {
    return (
      <div className="grid gap-3 md:grid-cols-3">
        <Skeleton className="h-28 w-full rounded-lg" />
        <Skeleton className="h-28 w-full rounded-lg" />
        <Skeleton className="h-28 w-full rounded-lg" />
      </div>
    );
  }

  return (
    <div className="grid gap-3 lg:grid-cols-3">
      <section className="rounded-lg border bg-background p-3">
        <div className="mb-3">
          <h3 className="text-sm font-semibold">Policy</h3>
          <p className="text-xs text-muted-foreground">Workspace-level runtime routing defaults.</p>
        </div>
        <div className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <span className="text-xs">Shared pool</span>
            <Switch checked={sharedPoolEnabled} onCheckedChange={setSharedPoolEnabled} />
          </div>
          <div className="flex items-center justify-between gap-3">
            <span className="text-xs">Allow Modal</span>
            <Switch checked={allowModal} onCheckedChange={setAllowModal} />
          </div>
          <div className="flex items-center justify-between gap-3">
            <span className="text-xs">Auto-stop</span>
            <Switch checked={autoStopEnabled} onCheckedChange={setAutoStopEnabled} />
          </div>
          <Button size="sm" className="w-full" onClick={handleSavePolicy} disabled={updatePolicy.isPending}>
            {updatePolicy.isPending ? "Saving..." : "Save Policy"}
          </Button>
        </div>
      </section>

      <section className="rounded-lg border bg-background p-3">
        <div className="mb-3">
          <h3 className="text-sm font-semibold">Billing</h3>
          <p className="text-xs text-muted-foreground">Managed-runtime pricing controls.</p>
        </div>
        <div className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <span className="text-xs">Billing enabled</span>
            <Switch checked={billingEnabled} onCheckedChange={setBillingEnabled} />
          </div>
          <Input value={sharedHourlyRate} onChange={(e) => setSharedHourlyRate(e.target.value)} placeholder="Shared cents/hour" />
          <Input value={privateHourlyRate} onChange={(e) => setPrivateHourlyRate(e.target.value)} placeholder="Private cents/hour" />
          <Input value={tokenMarkupPercent} onChange={(e) => setTokenMarkupPercent(e.target.value)} placeholder="Token markup %" />
          <Button size="sm" className="w-full" onClick={handleSaveBilling} disabled={updateBilling.isPending}>
            {updateBilling.isPending ? "Saving..." : "Save Billing"}
          </Button>
        </div>
      </section>

      <section className="rounded-lg border bg-background p-3">
        <div className="mb-3">
          <h3 className="text-sm font-semibold">Leases</h3>
          <p className="text-xs text-muted-foreground">Dedicated runtime reservations and lifecycle controls.</p>
        </div>
        <div className="space-y-2">
          <Input value={leaseName} onChange={(e) => setLeaseName(e.target.value)} placeholder="New lease name" />
          <div className="grid grid-cols-2 gap-2">
            <Button
              size="sm"
              variant={leaseBackend === "modal" ? "default" : "outline"}
              onClick={() => setLeaseBackend("modal")}
              disabled
            >
              Modal
            </Button>
            <Button
              size="sm"
              variant={leaseBackend === "local" ? "default" : "outline"}
              onClick={() => setLeaseBackend("local")}
            >
              Local
            </Button>
          </div>
          <p className="text-[11px] text-muted-foreground">
            Managed backends are not wired yet. Local leases create the runtime entries agents can actually use.
          </p>
          <div className="grid grid-cols-2 gap-2">
            <Button
              size="sm"
              variant={leasePlacement === "private" ? "default" : "outline"}
              onClick={() => setLeasePlacement("private")}
            >
              Private
            </Button>
            <Button
              size="sm"
              variant={leasePlacement === "shared" ? "default" : "outline"}
              onClick={() => setLeasePlacement("shared")}
            >
              Shared
            </Button>
          </div>
          <Button size="sm" className="w-full" onClick={handleCreateLease} disabled={createLease.isPending}>
            {createLease.isPending ? "Creating..." : "Create Lease"}
          </Button>
          <div className="max-h-32 space-y-2 overflow-y-auto pt-1">
            {leases.length === 0 ? (
              <p className="text-xs text-muted-foreground">No runtime leases yet.</p>
            ) : (
              leases.slice(0, 5).map((lease) => {
                const needsRepair = lease.backend === "local" && lease.state === "active" && !lease.runtime_id;
                const canStart = lease.state !== "active" || needsRepair;

                return (
                  <div key={lease.id} className="rounded-md border px-2 py-2">
                    <div className="flex items-center justify-between gap-2">
                      <div className="min-w-0">
                        <div className="truncate text-xs font-medium">{lease.name}</div>
                        <div className="text-[11px] text-muted-foreground">
                          {lease.backend} / {lease.state}
                          {needsRepair ? " / repair needed" : ""}
                        </div>
                      </div>
                      <div className="flex gap-1">
                        {canStart ? (
                          <Button size="sm" variant="outline" onClick={() => startLease.mutate(lease.id)}>
                            {needsRepair ? "Repair" : "Start"}
                          </Button>
                        ) : (
                          <Button size="sm" variant="outline" onClick={() => stopLease.mutate(lease.id)}>
                            Stop
                          </Button>
                        )}
                        <Button size="sm" variant="ghost" onClick={() => deleteLease.mutate(lease.id)}>
                          Delete
                        </Button>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>
      </section>
    </div>
  );
}
