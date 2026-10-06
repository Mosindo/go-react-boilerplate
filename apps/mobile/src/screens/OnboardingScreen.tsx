import React, { useEffect, useState } from "react";
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { saveInterests, saveLocation, saveProfile, savePreferences } from "../api/dating";
import type { OnboardingStep, Preferences } from "../api/types";
import { InterestPicker } from "../components/forms/InterestPicker";
import { LocationStep } from "../components/forms/LocationStep";
import { PhotoManager } from "../components/forms/PhotoManager";
import { PreferencesForm } from "../components/forms/PreferencesForm";
import { ProfileBasicsForm, type BasicsValues } from "../components/forms/ProfileBasicsForm";
import { useAuth } from "../hooks/useAuth";
import { describeError } from "../lib/errors";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Input, Notice, Text, radii, spacing, useTheme } from "../shared/ui";
import { finishOnboardingFlow, startOnboardingFlow } from "../state/onboardingFlow";

type Step = OnboardingStep | "about";

const STEPS: { key: Step; title: string; subtitle: string }[] = [
  { key: "profile", title: "Faisons connaissance", subtitle: "Ces informations apparaîtront sur votre profil." },
  { key: "location", title: "Où êtes-vous ?", subtitle: "Pour rencontrer des personnes près de chez vous." },
  { key: "preferences", title: "Qui souhaitez-vous rencontrer ?", subtitle: "Vous pourrez changer cela à tout moment." },
  { key: "about", title: "Parlez-nous de vous", subtitle: "Facultatif, mais les profils complets ont plus de matchs." },
  { key: "photos", title: "Ajoutez vos photos", subtitle: "Au moins une photo pour être visible." }
];

function firstIncompleteStep(missing: OnboardingStep[]): number {
  const index = STEPS.findIndex((step) => step.key !== "about" && missing.includes(step.key as OnboardingStep));
  return index === -1 ? STEPS.length - 1 : index;
}

export default function OnboardingScreen() {
  const { colors } = useTheme();
  const { self, refreshSelf, signOut } = useAuth();
  const [step, setStep] = useState(() => firstIncompleteStep(self?.onboarding.missing ?? ["profile"]));
  const [bio, setBio] = useState(self?.profile?.bio ?? "");
  const [interestIds, setInterestIds] = useState<number[]>(self?.interests.map((interest) => interest.id) ?? []);
  const [error, setError] = useState<string | null>(null);

  // Keep the flow on screen until "Terminer", even once the server says it is complete.
  useEffect(() => {
    startOnboardingFlow();
  }, []);

  const advance = async () => {
    await refreshSelf();
    setError(null);
    setStep((current) => Math.min(current + 1, STEPS.length - 1));
  };

  const run = useMutation({
    mutationFn: async (action: () => Promise<unknown>) => {
      await action();
    },
    onSuccess: advance,
    onError: (err) => setError(describeError(err))
  });

  const current = STEPS[step];
  const photos = self?.photos ?? [];
  const profile = self?.profile ?? null;

  const saveBasics = (values: BasicsValues) => run.mutate(() => saveProfile({ ...values, bio: profile?.bio ?? "" }));

  const saveAbout = () =>
    run.mutate(async () => {
      if (profile) {
        await saveProfile({ firstName: profile.firstName, birthdate: profile.birthdate, gender: profile.gender, bio: bio.trim() });
      }
      await saveInterests(interestIds);
    });

  return (
    <SafeAreaLayout>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
        <View style={styles.top}>
          {step > 0 ? (
            <Pressable accessibilityLabel="Étape précédente" accessibilityRole="button" hitSlop={12} onPress={() => setStep(step - 1)}>
              <Text tone="primary" weight="semibold">
                ‹ Retour
              </Text>
            </Pressable>
          ) : (
            <Pressable accessibilityRole="button" hitSlop={12} onPress={() => void signOut()}>
              <Text tone="muted">Se déconnecter</Text>
            </Pressable>
          )}
          <Text tone="muted" variant="label">
            Étape {step + 1} sur {STEPS.length}
          </Text>
        </View>
        <View style={[styles.track, { backgroundColor: colors.surfaceSubtle }]}>
          <View style={[styles.progress, { backgroundColor: colors.primary, width: `${((step + 1) / STEPS.length) * 100}%` }]} />
        </View>

        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled" testID={`onboarding-${current.key}`}>
          <View style={styles.titles}>
            <Text accessibilityRole="header" variant="title">
              {current.title}
            </Text>
            <Text tone="muted">{current.subtitle}</Text>
          </View>

          {current.key === "profile" ? (
            <ProfileBasicsForm busy={run.isPending} error={error} initial={profile} onSubmit={saveBasics} submitLabel="Continuer" />
          ) : null}

          {current.key === "location" ? (
            <LocationStep
              busy={run.isPending}
              currentCity={profile?.hasLocation ? profile.city : undefined}
              error={error}
              onLocated={(location) => run.mutate(() => saveLocation(location))}
              submitLabel={profile?.hasLocation ? "Mettre à jour ma position" : "Utiliser ma position"}
            />
          ) : null}
          {current.key === "location" && profile?.hasLocation ? (
            <Button label="Continuer" onPress={() => void advance()} variant="outline" />
          ) : null}

          {current.key === "preferences" ? (
            <PreferencesForm
              busy={run.isPending}
              error={error}
              initial={self?.preferences}
              onSubmit={(values: Preferences) => run.mutate(() => savePreferences(values))}
              submitLabel="Continuer"
            />
          ) : null}

          {current.key === "about" ? (
            <View style={styles.stack}>
              <Input
                helperText={`${bio.length}/500`}
                label="À propos de moi"
                maxLength={500}
                multiline
                onChangeText={setBio}
                placeholder="Quelques mots pour briser la glace…"
                testID="onboarding-bio"
                value={bio}
              />
              <Text tone="secondary" variant="label" weight="semibold">
                Centres d'intérêt
              </Text>
              <InterestPicker onChange={setInterestIds} selectedIds={interestIds} />
              {error ? <Notice message={error} tone="danger" /> : null}
              <Button label="Continuer" loading={run.isPending} onPress={saveAbout} testID="onboarding-about-next" />
              <Button label="Passer" onPress={() => void advance()} variant="ghost" />
            </View>
          ) : null}

          {current.key === "photos" ? (
            <View style={styles.stack}>
              <PhotoManager photos={photos} />
              <Button
                disabled={photos.length === 0}
                label="Terminer et découvrir"
                onPress={() => finishOnboardingFlow()}
                testID="onboarding-finish"
              />
            </View>
          ) : null}
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  top: {
    flexDirection: "row",
    justifyContent: "space-between",
    alignItems: "center",
    paddingHorizontal: spacing.xl,
    paddingTop: spacing.md,
    paddingBottom: spacing.sm
  },
  track: { height: 4, marginHorizontal: spacing.xl, borderRadius: radii.pill, overflow: "hidden" },
  progress: { height: 4, borderRadius: radii.pill },
  content: { gap: spacing.xl, padding: spacing.xl, paddingBottom: spacing.xxxl, maxWidth: 520, width: "100%", alignSelf: "center" },
  titles: { gap: spacing.xs },
  stack: { gap: spacing.lg }
});
