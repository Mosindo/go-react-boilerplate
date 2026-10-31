import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { savePreferences, saveProfile } from "../api/dating";
import type { OwnProfile, Preferences, ProfileInput } from "../api/types";
import { PhotoManager } from "../components/PhotoManager";
import { PreferencesForm } from "../components/PreferencesForm";
import { ProfileForm, toInput } from "../components/ProfileForm";
import { queryKeys } from "../lib/queryKeys";
import { Screen } from "../shared/layout";
import { Button, Notice, Text, colors, radii, spacing } from "../shared/ui";

type Step = "basics" | "photos" | "about" | "location" | "preferences";
const STEPS: Step[] = ["basics", "photos", "about", "location", "preferences"];

const COPY: Record<Step, { title: string; subtitle: string }> = {
  basics: { title: "Faisons connaissance", subtitle: "Votre prénom et votre genre." },
  photos: { title: "Vos plus belles photos", subtitle: "Au moins une photo est nécessaire. La première sera votre photo principale." },
  about: { title: "Parlez-nous de vous", subtitle: "Une courte bio et vos centres d'intérêt aident à briser la glace." },
  location: { title: "Où êtes-vous ?", subtitle: "Pour vous proposer des personnes proches de vous." },
  preferences: { title: "Qui souhaitez-vous rencontrer ?", subtitle: "Vous pourrez modifier ces préférences à tout moment." }
};

type OnboardingScreenProps = { profile: OwnProfile };

/** Guided profile creation. Nothing is saved server-side until the last step. */
export default function OnboardingScreen({ profile }: OnboardingScreenProps) {
  const queryClient = useQueryClient();
  const [stepIndex, setStepIndex] = useState(0);
  const [draft, setDraft] = useState<ProfileInput>(() => toInput(profile));
  const [prefs, setPrefs] = useState<Preferences>(profile.preferences);
  const [error, setError] = useState<string | null>(null);
  const step = STEPS[stepIndex] ?? "basics";
  const last = stepIndex === STEPS.length - 1;

  const finish = useMutation({
    mutationFn: async () => {
      await saveProfile(draft);
      return savePreferences(prefs);
    },
    onSuccess: (saved) => queryClient.setQueryData(queryKeys.profile, saved),
    onError: (err) => setError(err instanceof ApiError ? err.message : "Enregistrement impossible. Réessayez.")
  });

  const validate = (): string | null => {
    if (step === "basics") {
      if (!draft.firstName.trim()) return "Indiquez votre prénom.";
      if (!draft.gender) return "Choisissez votre genre.";
    }
    if (step === "photos" && profile.photos.length === 0) return "Ajoutez au moins une photo.";
    if (step === "preferences" && prefs.interestedIn.length === 0) return "Choisissez au moins un genre à rencontrer.";
    return null;
  };

  const next = () => {
    const message = validate();
    setError(message);
    if (message) return;
    if (last) finish.mutate();
    else setStepIndex((i) => i + 1);
  };

  return (
    <Screen scroll>
      <View style={styles.progress}>
        {STEPS.map((s, i) => (
          <View key={s} style={[styles.bar, i <= stepIndex ? styles.barDone : null]} />
        ))}
      </View>
      <View style={styles.header}>
        <Text tone="muted" variant="eyebrow" weight="bold">
          Étape {stepIndex + 1} sur {STEPS.length}
        </Text>
        <Text variant="title" weight="bold">
          {COPY[step].title}
        </Text>
        <Text tone="muted">{COPY[step].subtitle}</Text>
      </View>

      {step === "basics" ? <ProfileForm onChange={setDraft} sections={["identity"]} value={draft} /> : null}
      {step === "photos" ? <PhotoManager photos={profile.photos} /> : null}
      {step === "about" ? <ProfileForm onChange={setDraft} sections={["about", "interests"]} value={draft} /> : null}
      {step === "location" ? <ProfileForm onChange={setDraft} sections={["location"]} value={draft} /> : null}
      {step === "preferences" ? <PreferencesForm onChange={setPrefs} value={prefs} /> : null}

      {error ? <Notice title={error} tone="danger" /> : null}
      <View style={styles.actions}>
        {stepIndex > 0 ? <Button label="Retour" onPress={() => setStepIndex((i) => i - 1)} style={styles.flex} variant="outline" /> : null}
        <Button
          label={last ? "Terminer" : "Continuer"}
          loading={finish.isPending}
          onPress={next}
          style={styles.flex}
          testID="onboarding-next"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  progress: { flexDirection: "row", gap: spacing.xs, marginTop: spacing.md },
  bar: { flex: 1, height: 4, borderRadius: radii.pill, backgroundColor: colors.surfaceSubtle },
  barDone: { backgroundColor: colors.primary },
  header: { gap: spacing.xs },
  actions: { flexDirection: "row", gap: spacing.md },
  flex: { flex: 1 }
});
