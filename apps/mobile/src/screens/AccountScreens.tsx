import React, { useState } from "react";
import { Alert, StyleSheet, Switch, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { authApi, profileApi, safetyApi } from "../api/endpoints";
import type { ReportReason } from "../api/types";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesForm } from "../components/PreferencesForm";
import { ProfileForm } from "../components/ProfileForm";
import { validatePassword } from "../lib/validation";
import type { MainScreenProps } from "../navigation/types";
import { keys } from "../state/keys";
import { useMyProfile } from "../state/queries";
import { useSession } from "../state/session";
import { Button } from "../ui/Button";
import { Chip } from "../ui/Chip";
import { Input } from "../ui/Input";
import { Screen } from "../ui/Screen";
import { EmptyState, ErrorState, Loading } from "../ui/states";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";

function useProfileOrState() {
  const query = useMyProfile();
  const fallback = query.isLoading ? (
    <Screen>
      <Loading />
    </Screen>
  ) : query.isError || !query.data ? (
    <Screen>
      <ErrorState
        message={query.error?.message ?? "Profile unavailable."}
        onRetry={() => void query.refetch()}
      />
    </Screen>
  ) : null;
  return { profile: query.data, fallback };
}

export function EditProfileScreen({ navigation }: MainScreenProps<"EditProfile">) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { profile, fallback } = useProfileOrState();
  const save = useMutation({
    mutationFn: profileApi.save,
    onSuccess: (saved) => {
      queryClient.setQueryData(keys.profile, saved);
      toast.show("Profile saved.", "success");
      navigation.goBack();
    },
    onError: (e: Error) => toast.show(e.message, "error")
  });
  if (fallback || !profile) return fallback;
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <ProfileForm
        initial={profile}
        submitLabel="Save"
        saving={save.isPending}
        onSubmit={(input) => save.mutate(input)}
      />
    </Screen>
  );
}

export function PhotosScreen() {
  const { profile, fallback } = useProfileOrState();
  if (fallback || !profile) return fallback;
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <Text tone="muted">
        Tap a photo to make it your main photo, replace it or delete it. At least one photo is
        required.
      </Text>
      <PhotoManager photos={profile.photos} />
    </Screen>
  );
}

export function PreferencesScreen({ navigation }: MainScreenProps<"Preferences">) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { profile, fallback } = useProfileOrState();
  const save = useMutation({
    mutationFn: profileApi.savePreferences,
    onSuccess: (saved) => {
      queryClient.setQueryData(keys.profile, saved);
      toast.show("Preferences saved.", "success");
      navigation.goBack();
    },
    onError: (e: Error) => toast.show(e.message, "error")
  });
  if (fallback || !profile) return fallback;
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <PreferencesForm
        initial={profile.preferences}
        saving={save.isPending}
        submitLabel="Save"
        onSubmit={(v) => save.mutate(v)}
      />
    </Screen>
  );
}

export function SettingsScreen({ navigation }: MainScreenProps<"Settings">) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const { signOut } = useSession();
  const { profile, fallback } = useProfileOrState();
  const privacy = useMutation({
    mutationFn: profileApi.savePrivacy,
    onSuccess: (saved) => queryClient.setQueryData(keys.profile, saved),
    onError: (e: Error) => toast.show(e.message, "error")
  });
  if (fallback || !profile) return fallback;
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <ToggleRow
        label="Show me in discovery"
        hint="Turn off to pause your profile. Your matches can still chat with you."
        value={profile.discoverable}
        onChange={(v) => privacy.mutate({ discoverable: v })}
        testID="toggle-discoverable"
      />
      <ToggleRow
        label="Show my distance"
        hint="Others see an approximate distance, never your location."
        value={profile.showDistance}
        onChange={(v) => privacy.mutate({ showDistance: v })}
        testID="toggle-distance"
      />
      <Button
        label="Blocked people"
        variant="secondary"
        onPress={() => navigation.navigate("Blocked")}
      />
      <Button
        label="Change password"
        variant="secondary"
        onPress={() => navigation.navigate("ChangePassword")}
      />
      <Button
        label="Sign out"
        variant="secondary"
        onPress={() => void signOut()}
        testID="sign-out"
      />
      <Button
        label="Delete my account"
        variant="danger"
        onPress={() => navigation.navigate("DeleteAccount")}
      />
    </Screen>
  );
}

function ToggleRow({
  label,
  hint,
  value,
  onChange,
  testID
}: {
  label: string;
  hint: string;
  value: boolean;
  onChange: (v: boolean) => void;
  testID: string;
}) {
  return (
    <View style={styles.toggle}>
      <View style={styles.flex}>
        <Text variant="label">{label}</Text>
        <Text variant="caption" tone="muted">
          {hint}
        </Text>
      </View>
      <Switch testID={testID} accessibilityLabel={label} value={value} onValueChange={onChange} />
    </View>
  );
}

export function BlockedScreen() {
  const queryClient = useQueryClient();
  const toast = useToast();
  const query = useQuery({ queryKey: keys.blocked, queryFn: safetyApi.blocked });
  const unblock = useMutation({
    mutationFn: safetyApi.unblock,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: keys.blocked }),
    onError: (e: Error) => toast.show(e.message, "error")
  });
  if (query.isLoading)
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  if (query.isError)
    return (
      <Screen>
        <ErrorState message={query.error.message} onRetry={() => void query.refetch()} />
      </Screen>
    );
  const blocks = query.data ?? [];
  if (blocks.length === 0)
    return (
      <Screen>
        <EmptyState title="No one blocked" message="People you block will appear here." />
      </Screen>
    );
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      {blocks.map((b) => (
        <View key={b.userId} style={styles.toggle}>
          <Text variant="label" style={styles.flex}>
            {b.firstName || "Former member"}
          </Text>
          <Button label="Unblock" variant="secondary" onPress={() => unblock.mutate(b.userId)} />
        </View>
      ))}
    </Screen>
  );
}

export function ChangePasswordScreen({ navigation }: MainScreenProps<"ChangePassword">) {
  const toast = useToast();
  const { signOut } = useSession();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [error, setError] = useState<string | null>(null);
  const change = useMutation({
    mutationFn: () => authApi.changePassword(current, next),
    onSuccess: () => {
      toast.show("Password changed. Please sign in again.", "success");
      void signOut();
    },
    onError: (e: Error) => setError(e.message)
  });
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <Input label="Current password" value={current} onChangeText={setCurrent} secureTextEntry />
      <Input
        label="New password"
        value={next}
        onChangeText={setNext}
        secureTextEntry
        hint="At least 8 characters."
      />
      {error ? (
        <Text tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button
        label="Change password"
        loading={change.isPending}
        onPress={() => {
          const problem = validatePassword(next);
          setError(problem);
          if (!problem) change.mutate();
        }}
      />
      <Button label="Cancel" variant="ghost" onPress={() => navigation.goBack()} />
    </Screen>
  );
}

export function DeleteAccountScreen({ navigation }: MainScreenProps<"DeleteAccount">) {
  const { signOut } = useSession();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const remove = useMutation({
    mutationFn: () => authApi.deleteAccount(password),
    onSuccess: () => void signOut(),
    onError: (e: Error) => setError(e.message)
  });
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <Text variant="heading">Delete your account</Text>
      <Text tone="muted">
        This permanently removes your profile, photos, matches and messages. This cannot be undone.
      </Text>
      <Input
        label="Confirm with your password"
        value={password}
        onChangeText={setPassword}
        secureTextEntry
        testID="delete-password"
      />
      {error ? (
        <Text tone="danger" accessibilityRole="alert">
          {error}
        </Text>
      ) : null}
      <Button
        label="Delete my account permanently"
        variant="danger"
        loading={remove.isPending}
        disabled={!password}
        testID="delete-confirm"
        onPress={() =>
          Alert.alert("Are you sure?", "Your account and all your data will be deleted.", [
            { text: "Cancel", style: "cancel" },
            { text: "Delete", style: "destructive", onPress: () => remove.mutate() }
          ])
        }
      />
      <Button label="Cancel" variant="ghost" onPress={() => navigation.goBack()} />
    </Screen>
  );
}

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Fake profile" },
  { value: "harassment", label: "Harassment" },
  { value: "inappropriate_content", label: "Inappropriate content" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Seems underage" },
  { value: "other", label: "Other" }
];

export function ReportScreen({ navigation, route }: MainScreenProps<"Report">) {
  const { userId, name } = route.params;
  const toast = useToast();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason ?? "other", details.trim(), alsoBlock),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.matches });
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      toast.show("Thanks. Your report was sent to our moderators.", "success");
      navigation.popToTop();
    },
    onError: (e: Error) => toast.show(e.message, "error")
  });
  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <Text variant="heading">Report {name}</Text>
      <View style={styles.wrap}>
        {REASONS.map((r) => (
          <Chip
            key={r.value}
            label={r.label}
            selected={reason === r.value}
            onPress={() => setReason(r.value)}
          />
        ))}
      </View>
      <Input
        label="Details (optional)"
        value={details}
        onChangeText={setDetails}
        multiline
        maxLength={1000}
      />
      <ToggleRow
        label="Also block this person"
        hint="They will not be able to see or contact you."
        value={alsoBlock}
        onChange={setAlsoBlock}
        testID="report-block"
      />
      <Button
        label="Send report"
        disabled={!reason}
        loading={report.isPending}
        onPress={() => report.mutate()}
        testID="report-submit"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  toggle: { flexDirection: "row", alignItems: "center", gap: 12 },
  flex: { flex: 1 },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
