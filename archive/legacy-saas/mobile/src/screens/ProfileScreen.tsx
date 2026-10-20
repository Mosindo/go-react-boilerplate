import React, { useCallback, useEffect, useMemo, useState } from "react";
import { AppState, FlatList, Linking, StyleSheet, View } from "react-native";
import { type AuthUser } from "../api/auth";
import { useAuth } from "../hooks/useAuth";
import {
  createBillingCheckout,
  getBillingSubscription,
  listUsers,
  type BillingSubscription,
  type PlatformUser
} from "../api/platform";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Avatar, Badge, Button, Card, Notice, Text, colors, spacing } from "../shared/ui";
import { Header, ScreenContainer } from "../shared/layout";

type ProfileScreenProps = {
  user: AuthUser;
  token: string;
};

function formatMemberDate(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Unknown";
  }
  return date.toLocaleDateString();
}

function formatBillingDate(value?: string): string {
  if (!value) {
    return "Not available";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "Not available";
  }
  return date.toLocaleString();
}

function formatBillingStatus(status?: string): string {
  const value = (status ?? "inactive").replace(/_/g, " ").trim();
  if (!value) {
    return "Inactive";
  }
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function billingBadgeVariant(status?: string): "primary" | "success" | "warning" | "danger" | "muted" {
  switch ((status ?? "inactive").toLowerCase()) {
    case "active":
    case "trialing":
    case "paid":
      return "success";
    case "past_due":
    case "unpaid":
      return "warning";
    case "canceled":
    case "incomplete_expired":
      return "danger";
    case "open":
    case "checkout_open":
      return "primary";
    default:
      return "muted";
  }
}

function isSubscriptionActive(status?: string): boolean {
  switch ((status ?? "").toLowerCase()) {
    case "active":
    case "trialing":
    case "paid":
      return true;
    default:
      return false;
  }
}

export default function ProfileScreen({ user, token }: ProfileScreenProps) {
  const [users, setUsers] = useState<PlatformUser[]>([]);
  const [subscription, setSubscription] = useState<BillingSubscription | null>(null);
  const [loading, setLoading] = useState(true);
  const [billingLoading, setBillingLoading] = useState(true);
  const [startingCheckout, setStartingCheckout] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [billingError, setBillingError] = useState<string | null>(null);
  const [billingNotice, setBillingNotice] = useState<string | null>(null);
  const [awaitingCheckoutReturn, setAwaitingCheckoutReturn] = useState(false);
  const { logout, isLoggingOut } = useAuth();

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const nextUsers = await listUsers(token);
      setUsers(nextUsers.filter((candidate) => candidate.id !== user.id));
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "could not load members");
    } finally {
      setLoading(false);
    }
  }, [token, user.id]);

  const loadBilling = useCallback(async (options?: { silent?: boolean; afterCheckout?: boolean }) => {
    if (!options?.silent) {
      setBillingLoading(true);
    }
    setBillingError(null);

    try {
      const nextSubscription = await getBillingSubscription(token);
      setSubscription(nextSubscription);

      if (options?.afterCheckout) {
        if (isSubscriptionActive(nextSubscription.status)) {
          setBillingNotice("Subscription active. Your workspace billing has been updated.");
          setAwaitingCheckoutReturn(false);
        } else {
          setBillingNotice("Billing status refreshed. If payment is still in progress, return here in a moment.");
        }
      }
    } catch (loadError) {
      setBillingError(loadError instanceof Error ? loadError.message : "could not load billing");
    } finally {
      if (!options?.silent) {
        setBillingLoading(false);
      }
    }
  }, [token]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    void loadBilling();
  }, [loadBilling]);

  useEffect(() => {
    const subscription = AppState.addEventListener("change", (state) => {
      if (state === "active" && awaitingCheckoutReturn) {
        void loadBilling({ silent: true, afterCheckout: true });
      }
    });

    return () => {
      subscription.remove();
    };
  }, [awaitingCheckoutReturn, loadBilling]);

  async function onCheckout() {
    setStartingCheckout(true);
    setBillingError(null);
    setBillingNotice(null);

    try {
      const session = await createBillingCheckout(token);
      setAwaitingCheckoutReturn(true);
      setBillingNotice("Stripe Checkout opened in your browser. Return here after payment to refresh status.");
      await Linking.openURL(session.checkoutUrl);
    } catch (checkoutError) {
      setAwaitingCheckoutReturn(false);
      setBillingError(checkoutError instanceof Error ? checkoutError.message : "could not open checkout");
    } finally {
      setStartingCheckout(false);
    }
  }

  const directoryCountLabel = useMemo(() => {
    if (loading) {
      return "Loading members...";
    }
    if (users.length === 0) {
      return "No other members yet.";
    }
    return `${users.length} member${users.length > 1 ? "s" : ""} available`;
  }, [loading, users.length]);

  const subscriptionStatus = subscription?.status ?? "inactive";
  const nextRenewalLabel = subscription?.currentPeriodEnd
    ? `Current period ends ${formatBillingDate(subscription.currentPeriodEnd)}`
    : "No active renewal period yet.";
  const checkoutButtonLabel = isSubscriptionActive(subscriptionStatus) ? "Open billing checkout" : "Start subscription";

  return (
    <ScreenContainer contentMaxWidth={760} testID="profile-screen">
      <Header
        action={
          <Button
            label="Reload"
            onPress={() => {
              void load();
              void loadBilling();
            }}
            size="sm"
            testID="profile-reload-button"
            variant="outline"
          />
        }
        eyebrow="Workspace profile"
        style={styles.header}
        subtitle="Account identity, tenant context, and member directory."
        title="Profile"
      />

      <Card style={styles.card} variant="accent">
        <View style={styles.sessionHeader}>
          <Avatar name={user.email} size={56} />
          <View style={styles.sessionMeta}>
            <Text variant="heading" weight="bold">
              Session
            </Text>
            <Text tone="muted">{user.email}</Text>
          </View>
        </View>

        <Text style={styles.label} tone="muted" variant="eyebrow" weight="bold">
          Email
        </Text>
        <Text style={styles.value}>{user.email}</Text>

        <Text style={styles.label} tone="muted" variant="eyebrow" weight="bold">
          User ID
        </Text>
        <Text style={styles.value}>{user.id}</Text>

        <Text style={styles.label} tone="muted" variant="eyebrow" weight="bold">
          Organization
        </Text>
        <Text style={styles.value}>{user.organizationId}</Text>

        <Text style={styles.label} tone="muted" variant="eyebrow" weight="bold">
          Member since
        </Text>
        <Text style={styles.value}>{formatMemberDate(user.createdAt)}</Text>

        <Button
          fullWidth
          label={isLoggingOut ? "Logging out..." : "Logout"}
          onPress={logout}
          style={styles.button}
          testID="profile-logout-button"
          variant="outline"
        />
      </Card>

      <Card style={styles.card} variant="muted">
        <View style={styles.billingHeader}>
          <View style={styles.billingMeta}>
            <Text variant="heading" weight="bold">
              Billing
            </Text>
            <Text tone="muted">
              Stripe-hosted subscription checkout for this organization.
            </Text>
          </View>
          <Badge
            label={formatBillingStatus(subscriptionStatus)}
            size="sm"
            variant={billingBadgeVariant(subscriptionStatus)}
          />
        </View>

        <Text style={styles.label} tone="muted" variant="eyebrow" weight="bold">
          Provider
        </Text>
        <Text style={styles.value}>{subscription?.provider ?? "stripe"}</Text>

        <Text style={styles.label} tone="muted" variant="eyebrow" weight="bold">
          Status
        </Text>
        <Text style={styles.value}>{formatBillingStatus(subscriptionStatus)}</Text>

        <Text style={styles.label} tone="muted" variant="eyebrow" weight="bold">
          Renewal
        </Text>
        <Text style={styles.value}>{nextRenewalLabel}</Text>

        {billingNotice ? (
          <Notice
            description="Webhook updates can take a few seconds after payment confirmation."
            style={styles.billingNotice}
            title={billingNotice}
            tone={isSubscriptionActive(subscriptionStatus) ? "success" : "default"}
          />
        ) : null}

        {billingError ? (
          <ErrorView
            actionLabel="Retry"
            message={billingError}
            onAction={() => void loadBilling()}
            style={styles.billingError}
          />
        ) : null}

        {billingLoading ? (
          <LoadingView label="Loading billing..." style={styles.billingLoader} />
        ) : (
          <View style={styles.billingActions}>
            <Button
              fullWidth
              label={checkoutButtonLabel}
              loading={startingCheckout}
              onPress={onCheckout}
              testID="profile-billing-checkout-button"
            />
            <Button
              fullWidth
              label="Refresh billing"
              onPress={() => void loadBilling()}
              testID="profile-billing-refresh-button"
              variant="outline"
            />
          </View>
        )}
      </Card>

      <View style={styles.directoryHeader}>
        <View>
          <Text style={styles.sectionTitle} variant="heading" weight="bold">
            Directory
          </Text>
          <Text style={styles.directoryMeta} tone="muted">
            {directoryCountLabel}
          </Text>
        </View>
      </View>

      {error ? (
        <ErrorView actionLabel="Retry" message={error} onAction={() => void load()} style={styles.error} />
      ) : null}

      {loading ? (
        <LoadingView fullScreen label="Loading members..." style={styles.loaderWrap} />
      ) : (
        <FlatList
          contentContainerStyle={styles.list}
          data={users}
          keyExtractor={(item) => item.id}
          ListEmptyComponent={
            <EmptyView
              message="Invite teammates or create another test account to populate the directory."
              title="No team members yet"
            />
          }
          renderItem={({ item }) => (
            <Card padding="sm" style={styles.memberCard} variant="muted">
              <View style={styles.memberRow}>
                <Avatar name={item.email} size={40} />
                <View style={styles.memberTextWrap}>
                  <Text style={styles.memberEmail} variant="label" weight="bold">
                    {item.email}
                  </Text>
                  <Text style={styles.memberMeta} tone="muted">
                    Joined {formatMemberDate(item.createdAt)}
                  </Text>
                </View>
              </View>
            </Card>
          )}
        />
      )}
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  header: {
    marginBottom: spacing.md
  },
  card: {
    marginBottom: spacing.lg
  },
  sessionHeader: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    marginBottom: spacing.md
  },
  sessionMeta: {
    flex: 1,
    gap: spacing.xs
  },
  sectionTitle: {
    color: colors.text,
    marginBottom: spacing.xs
  },
  label: {
    marginTop: spacing.sm
  },
  value: {
    marginTop: spacing.xs,
    color: colors.text
  },
  button: {
    marginTop: spacing.xl
  },
  billingHeader: {
    flexDirection: "row",
    alignItems: "flex-start",
    justifyContent: "space-between",
    gap: spacing.md,
    marginBottom: spacing.sm
  },
  billingMeta: {
    flex: 1,
    gap: spacing.xs
  },
  billingNotice: {
    marginTop: spacing.lg
  },
  billingError: {
    marginTop: spacing.lg
  },
  billingLoader: {
    marginTop: spacing.lg
  },
  billingActions: {
    marginTop: spacing.lg,
    gap: spacing.sm
  },
  directoryHeader: {
    marginBottom: spacing.sm
  },
  directoryMeta: {
    color: colors.textMuted
  },
  error: {
    marginBottom: spacing.sm
  },
  loaderWrap: {
    flex: 1
  },
  list: {
    paddingBottom: spacing.xxxl
  },
  memberCard: {
    marginBottom: spacing.sm
  },
  memberRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md
  },
  memberTextWrap: {
    flex: 1,
    gap: spacing.xs
  },
  memberEmail: {
    color: colors.text
  },
  memberMeta: {
    color: colors.textMuted
  },
  empty: {
    marginTop: spacing.xxl
  }
});
