import React, { useState } from "react";
import { StyleSheet, Switch, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { deleteAccount, errorMessage, getOwnProfile, listBlocks, queryKeys, savePrivacy, unblockUser } from "../api/platform";
import { Screen } from "../components/Screen";
import { useAuth } from "../hooks/useAuth";
import { LoadingView, showToast } from "../shared/feedback";
import { Button, Card, Input, Notice, Text, colors, spacing } from "../shared/ui";
import { showDialog } from "../shared/dialog";

function ToggleRow({ hint, label, onChange, testID, value }: { label: string; hint: string; value: boolean; onChange: (v: boolean) => void; testID: string }) {
  return (
    <View style={styles.toggle}>
      <View style={styles.toggleCopy}>
        <Text weight="semibold">{label}</Text>
        <Text tone="muted" variant="caption">
          {hint}
        </Text>
      </View>
      <Switch accessibilityLabel={label} onValueChange={onChange} testID={testID} thumbColor={colors.backgroundElevated} trackColor={{ true: colors.primary, false: colors.borderStrong }} value={value} />
    </View>
  );
}

export default function SettingsScreen() {
  const client = useQueryClient();
  const { endSession, logout } = useAuth();
  const profile = useQuery({ queryKey: queryKeys.profile, queryFn: getOwnProfile });
  const blocks = useQuery({ queryKey: queryKeys.blocks, queryFn: listBlocks });
  const [confirming, setConfirming] = useState(false);
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);

  const privacy = useMutation({
    mutationFn: (v: { isVisible: boolean; showDistance: boolean }) => savePrivacy(v.isVisible, v.showDistance),
    onSuccess: () => client.invalidateQueries({ queryKey: queryKeys.profile }),
    onError: (e) => showToast(errorMessage(e), { tone: "error" })
  });
  const unblock = useMutation({
    mutationFn: unblockUser,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.blocks });
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      void client.invalidateQueries({ queryKey: queryKeys.discover });
    },
    onError: (e) => showToast(errorMessage(e), { tone: "error" })
  });
  const remove = useMutation({
    mutationFn: () => deleteAccount(password),
    onSuccess: async () => {
      await endSession();
    },
    onError: (e) => setError(errorMessage(e))
  });

  if (!profile.data) {
    return <LoadingView fullScreen label="Loading…" />;
  }
  const p = profile.data;

  const askDelete = () =>
    showDialog("Delete your account?", "Your profile, photos, matches and messages are erased permanently. This cannot be undone.", [
      { text: "Cancel", style: "cancel" },
      { text: "Continue", style: "destructive", onPress: () => setConfirming(true) }
    ]);

  return (
    <Screen testID="settings-screen">
      <Card padding="md" style={styles.card}>
        <Text variant="heading" weight="bold">
          Privacy
        </Text>
        <ToggleRow
          hint="Turn off to pause your profile. Your matches can still talk to you."
          label="Show me in discovery"
          onChange={(v) => privacy.mutate({ isVisible: v, showDistance: p.showDistance })}
          testID="toggle-visible"
          value={p.isVisible}
        />
        <ToggleRow
          hint="Others only ever see an approximate distance, never a location."
          label="Show my distance"
          onChange={(v) => privacy.mutate({ isVisible: p.isVisible, showDistance: v })}
          testID="toggle-distance"
          value={p.showDistance}
        />
      </Card>

      <Card padding="md" style={styles.card}>
        <Text variant="heading" weight="bold">
          Blocked people
        </Text>
        {(blocks.data ?? []).length === 0 ? <Text tone="muted">You have not blocked anyone.</Text> : null}
        {(blocks.data ?? []).map((b) => (
          <View key={b.id} style={styles.blockRow}>
            <Text weight="semibold">{b.firstName || "Former member"}</Text>
            <Button label="Unblock" onPress={() => unblock.mutate(b.id)} size="sm" variant="outline" />
          </View>
        ))}
      </Card>

      <Card padding="md" style={styles.card}>
        <Text variant="heading" weight="bold">
          Account
        </Text>
        <Button label="Sign out" onPress={() => void logout()} testID="settings-logout" variant="outline" />
        {confirming ? (
          <View style={styles.card}>
            {error ? <Notice description={error} title="Could not delete" tone="danger" /> : null}
            <Input autoCapitalize="none" label="Confirm with your password" onChangeText={setPassword} secureTextEntry testID="delete-password" value={password} />
            <Button disabled={!password} label="Permanently delete my account" loading={remove.isPending} onPress={() => remove.mutate()} testID="delete-confirm" variant="destructive" />
            <Button label="Keep my account" onPress={() => setConfirming(false)} variant="ghost" />
          </View>
        ) : (
          <Button label="Delete my account" onPress={askDelete} testID="settings-delete" variant="ghost" textStyle={{ color: colors.danger }} />
        )}
      </Card>
      <Text style={styles.free} tone="muted" variant="caption">
        Amora is free. No subscriptions, no boosts, no limits on likes.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  card: { gap: spacing.md },
  toggle: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  toggleCopy: { flex: 1, gap: 2 },
  blockRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  free: { textAlign: "center" }
});
