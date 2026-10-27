import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import type { MyProfile, ProfileInput } from "../api/types";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesForm } from "../components/PreferencesForm";
import { ProfileForm } from "../components/ProfileForm";
import { keys } from "../state/keys";
import { useSession } from "../state/session";
import { Button } from "../ui/Button";
import { Screen } from "../ui/Screen";
import { Text } from "../ui/Text";
import { useToast } from "../ui/Toast";
import { usePalette } from "../theme";

const STEPS = ["About you", "Photos", "Who you'd like to meet"] as const;

/** Three-step profile creation. Progress is derived from server state so it survives restarts. */
export function OnboardingScreen({
  profile,
  onFinish
}: {
  profile: MyProfile | null;
  onFinish: () => void;
}) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const p = usePalette();
  const { signOut } = useSession();
  const [step, setStep] = useState(profile ? 1 : 0);

  const save = useMutation({
    mutationFn: (input: ProfileInput) => profileApi.save(input),
    onSuccess: (saved) => {
      queryClient.setQueryData(keys.profile, saved);
      setStep(1);
    },
    onError: (e: Error) => toast.show(e.message, "error")
  });
  const prefs = useMutation({
    mutationFn: profileApi.savePreferences,
    onSuccess: (saved) => {
      queryClient.setQueryData(keys.profile, saved);
      onFinish();
    },
    onError: (e: Error) => toast.show(e.message, "error")
  });

  const photos = profile?.photos ?? [];

  return (
    <Screen scroll>
      <View style={styles.progress} accessibilityLabel={`Step ${step + 1} of ${STEPS.length}`}>
        {STEPS.map((label, i) => (
          <View
            key={label}
            style={[styles.bar, { backgroundColor: i <= step ? p.primary : p.border }]}
          />
        ))}
      </View>
      <Text variant="title" accessibilityRole="header">
        {STEPS[step]}
      </Text>

      {step === 0 ? (
        <ProfileForm
          initial={profile}
          submitLabel="Continue"
          saving={save.isPending}
          onSubmit={(input) => save.mutate(input)}
        />
      ) : null}

      {step === 1 ? (
        <>
          <Text tone="muted">
            Add at least one photo. The first one is your main photo; you can reorder later.
          </Text>
          <PhotoManager photos={photos} />
          <Button
            label="Continue"
            disabled={photos.length === 0}
            onPress={() => setStep(2)}
            testID="onboarding-photos-next"
          />
          <Button label="Back" variant="ghost" onPress={() => setStep(0)} />
        </>
      ) : null}

      {step === 2 && profile ? (
        <PreferencesForm
          initial={profile.preferences}
          saving={prefs.isPending}
          submitLabel="Start meeting people"
          onSubmit={(value) => prefs.mutate(value)}
        />
      ) : null}
      {step === 2 ? <Button label="Back" variant="ghost" onPress={() => setStep(1)} /> : null}
      <Button label="Sign out" variant="ghost" onPress={() => void signOut()} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: 6, marginTop: 8 },
  bar: { flex: 1, height: 4, borderRadius: 2 }
});
