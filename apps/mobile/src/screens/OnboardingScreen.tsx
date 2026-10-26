import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi, type OwnProfile, type ProfileInput } from "../api";
import { useAuth } from "../auth/AuthContext";
import { Button } from "../components/Button";
import { LocationPicker } from "../components/LocationPicker";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesForm } from "../components/PreferencesForm";
import { ProfileForm } from "../components/ProfileForm";
import { Screen } from "../components/Screen";
import { Text } from "../components/Text";
import { queryKeys } from "../lib/queryKeys";
import { radius, spacing, useTheme } from "../theme";

type Step = "profile" | "preferences" | "photos" | "location";
const STEPS: Step[] = ["profile", "preferences", "photos", "location"];

const TITLES: Record<Step, { title: string; subtitle: string }> = {
  profile: { title: "Faisons connaissance", subtitle: "Ces informations apparaîtront sur votre profil." },
  preferences: {
    title: "Qui souhaitez-vous rencontrer ?",
    subtitle: "Vous pourrez modifier ces critères à tout moment."
  },
  photos: {
    title: "Vos plus belles photos",
    subtitle: "Ajoutez au moins une photo (6 maximum). La première sera votre photo principale."
  },
  location: { title: "Où êtes-vous ?", subtitle: "Pour vous présenter des personnes proches de vous." }
};

function firstStep(profile: OwnProfile | null | undefined): Step {
  if (!profile) return "profile";
  if (profile.missing.includes("preferences")) return "preferences";
  if (profile.missing.includes("photos")) return "photos";
  return "location";
}

/** Guided profile creation; each step is saved on the server so it can be resumed. */
export function OnboardingScreen({ profile }: { profile: OwnProfile | null | undefined }) {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const { signOut } = useAuth();
  const [step, setStep] = useState<Step>(() => firstStep(profile));
  const [serverError, setServerError] = useState<string | null>(null);

  const afterSave = (saved: OwnProfile, next: Step) => {
    queryClient.setQueryData(queryKeys.profile, saved);
    setServerError(null);
    setStep(next);
  };

  const saveProfile = useMutation({
    mutationFn: (input: ProfileInput) => profileApi.save(input),
    onSuccess: (saved) => afterSave(saved, firstStep(saved)),
    onError: (err) => setServerError(err instanceof Error ? err.message : "Enregistrement impossible.")
  });

  const savePrefs = useMutation({
    mutationFn: profileApi.savePreferences,
    onSuccess: (saved) => afterSave(saved, "photos"),
    onError: (err) => setServerError(err instanceof Error ? err.message : "Enregistrement impossible.")
  });

  const photosCount = profile?.photos.length ?? 0;
  const index = STEPS.indexOf(step);
  const { title, subtitle } = TITLES[step];

  const finish = () => void queryClient.invalidateQueries({ queryKey: queryKeys.profile });

  return (
    <Screen scroll contentStyle={styles.content}>
      <View style={styles.header}>
        <View
          style={styles.progress}
          accessibilityRole="progressbar"
          accessibilityLabel={`Étape ${index + 1} sur ${STEPS.length}`}
        >
          {STEPS.map((s, i) => (
            <View
              key={s}
              style={[styles.bar, { backgroundColor: i <= index ? theme.primary : theme.border }]}
            />
          ))}
        </View>
        <Text variant="title">{title}</Text>
        <Text muted>{subtitle}</Text>
      </View>

      {step === "profile" ? (
        <ProfileForm
          initial={profile}
          submitLabel="Continuer"
          loading={saveProfile.isPending}
          serverError={serverError}
          onSubmit={(input) => saveProfile.mutate(input)}
        />
      ) : null}

      {step === "preferences" ? (
        <PreferencesForm
          initial={profile?.preferences}
          submitLabel="Continuer"
          loading={savePrefs.isPending}
          onSubmit={(p) => savePrefs.mutate(p)}
        />
      ) : null}

      {step === "photos" ? (
        <View style={styles.block}>
          <PhotoManager photos={profile?.photos ?? []} />
          <Button
            label="Continuer"
            disabled={photosCount === 0}
            onPress={() => setStep("location")}
            testID="photos-next"
          />
        </View>
      ) : null}

      {step === "location" ? (
        <View style={styles.block}>
          <LocationPicker initialCity={profile?.city} hasLocation={!!profile?.hasLocation} onSaved={finish} />
        </View>
      ) : null}

      {serverError && step !== "profile" ? (
        <Text color={theme.danger} accessibilityRole="alert">
          {serverError}
        </Text>
      ) : null}

      <View style={styles.footer}>
        {index > 0 ? (
          <Button
            label="Retour"
            variant="ghost"
            small
            onPress={() => setStep(STEPS[index - 1] ?? "profile")}
          />
        ) : null}
        <Button label="Se déconnecter" variant="ghost" small onPress={() => void signOut()} />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { maxWidth: 560, width: "100%", alignSelf: "center", gap: spacing.xl },
  header: { gap: spacing.sm },
  progress: { flexDirection: "row", gap: spacing.xs, marginBottom: spacing.md },
  bar: { flex: 1, height: 5, borderRadius: radius.pill },
  block: { gap: spacing.lg },
  footer: { flexDirection: "row", justifyContent: "space-between" }
});
