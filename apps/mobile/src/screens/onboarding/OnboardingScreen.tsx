import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../../api/endpoints";
import type { MyProfile, Preferences } from "../../api/types";
import { useAuth } from "../../auth/AuthProvider";
import { AppText } from "../../components/AppText";
import { Button } from "../../components/Button";
import { InterestsPicker } from "../../components/profile/InterestsPicker";
import { LocationCard } from "../../components/profile/LocationCard";
import { PhotoGrid } from "../../components/profile/PhotoGrid";
import { PreferencesForm } from "../../components/profile/PreferencesForm";
import { draftFromProfile, draftToInput, ProfileForm, validateDraft, validateIdentity, type DraftErrors, type ProfileDraft } from "../../components/profile/ProfileForm";
import { Screen } from "../../components/Screen";
import { LoadingState } from "../../components/States";
import { useInterests, useMyProfile } from "../../hooks/queries";
import { queryKeys } from "../../queryClient";
import { colors, radii, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { useOnboardingHold } from "../../navigation/onboardingHold";

const steps = ["identity", "about", "preferences", "interests", "photos", "location", "done"] as const;
type Step = (typeof steps)[number];

const titles: Record<Step, { title: string; subtitle: string }> = {
  identity: { title: "Faisons connaissance", subtitle: "Ces informations apparaissent sur ton profil (sauf ta date de naissance)." },
  about: { title: "Parle-nous de toi", subtitle: "Quelques mots suffisent pour lancer la conversation." },
  preferences: { title: "Qui veux-tu rencontrer ?", subtitle: "Tu pourras modifier tout ça à tout moment." },
  interests: { title: "Tes centres d'intérêt", subtitle: "Ils sont mis en avant quand vous en partagez." },
  photos: { title: "Tes photos", subtitle: "Ajoute au moins une photo où l'on voit bien ton visage." },
  location: { title: "Autour de toi", subtitle: "Pour te proposer des personnes proches." },
  done: { title: "Tout est prêt ✨", subtitle: "Ton profil est en ligne. Bonne découverte !" }
};

export default function OnboardingScreen() {
  const client = useQueryClient();
  const { signOut } = useAuth();
  const { setHold } = useOnboardingHold();
  const profileQuery = useMyProfile();
  const interestsQuery = useInterests();
  const [stepIndex, setStepIndex] = useState(0);
  const [draft, setDraft] = useState<ProfileDraft | null>(null);
  const [errors, setErrors] = useState<DraftErrors>({});
  const [prefs, setPrefs] = useState<Preferences | null>(null);
  const [interestIds, setInterestIds] = useState<number[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Keep the onboarding on screen until the member taps "finish", even once
  // the profile becomes complete server-side.
  useEffect(() => {
    setHold(true);
  }, [setHold]);

  const data = profileQuery.data;
  useEffect(() => {
    if (data && !draft) {
      setDraft(draftFromProfile(data.profile));
      setPrefs(data.preferences);
      setInterestIds(data.interests.map((i) => i.id));
    }
  }, [data, draft]);

  if (!data || !draft || !prefs) {
    return (
      <Screen>
        <LoadingState label="Préparation de ton profil…" />
      </Screen>
    );
  }

  const step = steps[stepIndex];
  const next = () => setStepIndex((i) => Math.min(i + 1, steps.length - 1));
  const back = () => setStepIndex((i) => Math.max(i - 1, 0));

  const run = async (action: () => Promise<void>) => {
    setSaving(true);
    setError(null);
    try {
      await action();
      next();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  };

  const onContinue = () => {
    setError(null);
    switch (step) {
      case "identity": {
        const found = validateIdentity(draft);
        setErrors(found);
        if (Object.keys(found).length === 0) next();
        return;
      }
      case "about": {
        const found = validateDraft(draft);
        setErrors(found);
        if (Object.keys(found).length > 0) return;
        void run(async () => {
          client.setQueryData(queryKeys.myProfile, await profileApi.save(draftToInput(draft)));
        });
        return;
      }
      case "preferences":
        void run(async () => {
          const saved = await profileApi.setPreferences(prefs);
          client.setQueryData<MyProfile>(queryKeys.myProfile, (d) => (d ? { ...d, preferences: saved } : d));
        });
        return;
      case "interests":
        void run(async () => {
          const saved = await profileApi.setInterests(interestIds);
          client.setQueryData<MyProfile>(queryKeys.myProfile, (d) => (d ? { ...d, interests: saved.interests } : d));
        });
        return;
      case "photos":
        if (data.photos.length === 0) {
          setError("Ajoute au moins une photo pour continuer.");
          return;
        }
        next();
        return;
      case "location":
        if (!data.profile?.hasLocation) {
          setError("Partage ta position approximative pour continuer.");
          return;
        }
        next();
        return;
      case "done":
        void client.invalidateQueries({ queryKey: queryKeys.discovery });
        setHold(false);
        return;
    }
  };

  const { title, subtitle } = titles[step];
  const progress = (stepIndex + 1) / steps.length;

  return (
    <Screen
      scroll
      edges={["top", "bottom"]}
      testID={`onboarding-${step}`}
      footer={
        <>
          {error ? (
            <AppText color={colors.danger} align="center" testID="onboarding-error">
              {error}
            </AppText>
          ) : null}
          <Button label={step === "done" ? "Découvrir des profils" : step === "interests" && interestIds.length === 0 ? "Passer" : "Continuer"} onPress={onContinue} loading={saving} testID="onboarding-continue" />
          {stepIndex > 0 && step !== "done" ? <Button label="Retour" variant="ghost" onPress={back} disabled={saving} /> : null}
          {stepIndex === 0 ? <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} /> : null}
        </>
      }
    >
      <View style={styles.progressTrack} accessibilityRole="progressbar" accessibilityValue={{ min: 0, max: steps.length, now: stepIndex + 1 }}>
        <View style={[styles.progressFill, { width: `${progress * 100}%` }]} />
      </View>
      <View style={styles.heading}>
        <AppText variant="title" accessibilityRole="header">
          {title}
        </AppText>
        <AppText color={colors.textMuted}>{subtitle}</AppText>
      </View>
      {step === "identity" ? <ProfileForm draft={draft} onChange={setDraft} errors={errors} sections={["identity"]} /> : null}
      {step === "about" ? <ProfileForm draft={draft} onChange={setDraft} errors={errors} sections={["about"]} /> : null}
      {step === "preferences" ? <PreferencesForm value={prefs} onChange={setPrefs} /> : null}
      {step === "interests" ? (
        interestsQuery.data ? <InterestsPicker catalog={interestsQuery.data} selected={interestIds} onChange={setInterestIds} /> : <LoadingState />
      ) : null}
      {step === "photos" ? <PhotoGrid photos={data.photos} /> : null}
      {step === "location" ? <LocationCard hasLocation={!!data.profile?.hasLocation} city={data.profile?.city ?? ""} /> : null}
      {step === "done" ? (
        <View style={styles.done}>
          <AppText color={colors.textMuted}>
            Rappel : Lueur est entièrement gratuit. Likes illimités, aucune option payante. Si quelqu’un te met mal à l’aise, tu peux le bloquer ou le signaler depuis son profil ou la conversation.
          </AppText>
        </View>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  progressTrack: { height: 4, borderRadius: radii.pill, backgroundColor: colors.surfaceRaised, overflow: "hidden", marginBottom: spacing.lg },
  progressFill: { height: 4, backgroundColor: colors.primary },
  heading: { gap: spacing.xs, marginBottom: spacing.xl },
  done: { padding: spacing.md, borderRadius: radii.md, backgroundColor: colors.surface, borderWidth: 1, borderColor: colors.border }
});
