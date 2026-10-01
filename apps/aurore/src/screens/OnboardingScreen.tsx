import React, { useState } from "react";
import { View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import { keys } from "../api/keys";
import type { SelfProfile } from "../api/types";
import { useAuth } from "../auth/AuthProvider";
import { useFeedback } from "../components/Feedback";
import { BasicsForm, InterestPicker, PhotoManager, PreferencesForm, ShareLocationButton, errorMessage } from "../components/forms";
import { Button, Screen, Text, TextField } from "../components/ui";
import { MAX_BIO } from "../lib/validation";
import { spacing, useTheme } from "../theme/theme";

const STEPS = ["Vous", "Vos envies", "Photos", "Votre style", "Position"] as const;

function startStep(p: SelfProfile): number {
  if (!p.exists) return 0;
  if (p.photos.length === 0) return 2;
  return 0;
}

export function OnboardingScreen({ profile, onDone }: { profile: SelfProfile; onDone: () => void }) {
  const t = useTheme();
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const { signOut } = useAuth();
  const [step, setStep] = useState(() => startStep(profile));
  const [interests, setInterests] = useState<number[]>(profile.interests.map((i) => i.id));
  const [bio, setBio] = useState(profile.bio);

  const saveStyle = useMutation({
    mutationFn: async () => {
      await profileApi.saveInterests(interests);
      return profileApi.save({
        firstName: profile.firstName,
        birthDate: profile.birthDate,
        gender: profile.gender as "man" | "woman" | "non_binary",
        bio: bio.trim(),
        city: profile.city
      });
    },
    onSuccess: (p) => {
      qc.setQueryData(keys.profile, p);
      setStep(4);
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const next = () => setStep((s) => Math.min(s + 1, STEPS.length - 1));

  return (
    <Screen scroll>
      <View style={{ flexDirection: "row", gap: 6, marginTop: spacing.lg, marginBottom: spacing.xl }} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View key={s} style={{ flex: 1, height: 5, borderRadius: 3, backgroundColor: i <= step ? t.primary : t.border }} />
        ))}
      </View>
      <Text variant="caption" muted style={{ marginBottom: spacing.xs }}>
        Étape {step + 1} sur {STEPS.length}
      </Text>

      {step === 0 ? (
        <>
          <Text variant="title" style={{ marginBottom: spacing.sm }}>Faisons connaissance</Text>
          <Text muted style={{ marginBottom: spacing.xl }}>Ces informations apparaîtront sur votre profil.</Text>
          <BasicsForm profile={profile.exists ? profile : undefined} submitLabel="Continuer" onSaved={next} />
        </>
      ) : null}

      {step === 1 ? (
        <>
          <Text variant="title" style={{ marginBottom: spacing.sm }}>Qui souhaitez-vous rencontrer ?</Text>
          <Text muted style={{ marginBottom: spacing.xl }}>Modifiable à tout moment dans votre profil.</Text>
          <PreferencesForm profile={profile} submitLabel="Continuer" onSaved={next} />
        </>
      ) : null}

      {step === 2 ? (
        <>
          <Text variant="title" style={{ marginBottom: spacing.sm }}>Ajoutez vos photos</Text>
          <Text muted style={{ marginBottom: spacing.xl }}>Au moins une photo est nécessaire pour découvrir des profils.</Text>
          <PhotoManager profile={profile} />
          <Button label="Continuer" onPress={next} disabled={profile.photos.length === 0} style={{ marginTop: spacing.xl }} testID="photos-continue" />
        </>
      ) : null}

      {step === 3 ? (
        <>
          <Text variant="title" style={{ marginBottom: spacing.sm }}>Ce qui vous ressemble</Text>
          <TextField label="Quelques mots sur vous" value={bio} onChangeText={setBio} multiline maxLength={MAX_BIO} hint={`${bio.length}/${MAX_BIO}`} placeholder="Ce que vous aimez, ce que vous cherchez…" testID="onboarding-bio" />
          <Text variant="label" style={{ marginBottom: spacing.sm }}>Centres d&apos;intérêt</Text>
          <InterestPicker selected={interests} onChange={setInterests} />
          <Button label="Continuer" onPress={() => saveStyle.mutate()} loading={saveStyle.isPending} style={{ marginTop: spacing.xl }} testID="style-continue" />
        </>
      ) : null}

      {step === 4 ? (
        <>
          <Text variant="title" style={{ marginBottom: spacing.sm }}>Près de chez vous</Text>
          <Text muted style={{ marginBottom: spacing.xl }}>
            Aurore utilise votre position uniquement pour trouver des personnes proches. Les autres voient une distance approximative, jamais votre position exacte, et nous n&apos;en conservons qu&apos;une version arrondie à environ 1 km.
          </Text>
          <ShareLocationButton variant="primary" onDone={onDone} />
          <Button label="Plus tard" variant="ghost" onPress={onDone} style={{ marginTop: spacing.sm }} testID="location-skip" />
        </>
      ) : null}

      <View style={{ flex: 1 }} />
      <View style={{ flexDirection: "row", justifyContent: "space-between", marginTop: spacing.xl }}>
        {step > 0 ? <Button label="Retour" variant="ghost" onPress={() => setStep((s) => s - 1)} /> : <View />}
        <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
      </View>
    </Screen>
  );
}
