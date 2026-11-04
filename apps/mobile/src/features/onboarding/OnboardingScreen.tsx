import React, { useState } from "react";
import { View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import type { OwnProfile } from "../../api/types";
import { keys } from "../../hooks/queries";
import { useSession } from "../../hooks/session";
import { Screen } from "../../shared/layout";
import { Button, Text, radius, spacing, useColors, useToast } from "../../shared/ui";
import { AboutForm } from "./AboutForm";
import { BasicsForm } from "./BasicsForm";
import { LocationStep } from "./LocationStep";
import { PhotosManager } from "./PhotosManager";
import { PreferencesForm } from "./PreferencesForm";

const STEPS = ["Vous", "Vos envies", "Position", "À propos", "Photos"] as const;

function initialStep(profile: OwnProfile): number {
  if (!profile.firstName) return 0;
  if (!profile.preferences) return 1;
  return 2;
}

export function OnboardingScreen({ profile }: { profile: OwnProfile }) {
  const qc = useQueryClient();
  const c = useColors();
  const toast = useToast();
  const { signOut } = useSession();
  const [step, setStep] = useState(() => initialStep(profile));
  const next = () => setStep((s) => Math.min(s + 1, STEPS.length - 1));
  const sync = (p: OwnProfile) => qc.setQueryData(keys.profile, p);
  const onError = (err: unknown) => toast.show(errorMessage(err), "error");

  const saveBasics = useMutation({
    mutationFn: (v: { firstName: string; birthDate?: string; gender: OwnProfile["gender"] }) =>
      profileApi.save({ firstName: v.firstName, birthDate: v.birthDate, gender: v.gender }),
    onSuccess: (p) => { sync(p); next(); },
    onError,
  });
  const savePrefs = useMutation({ mutationFn: profileApi.savePreferences, onSuccess: (p) => { sync(p); next(); }, onError });
  const saveAbout = useMutation({
    mutationFn: (v: { bio: string; city: string; interests: string[] }) =>
      profileApi.save({ firstName: profile.firstName, gender: profile.gender, bio: v.bio, city: v.city || undefined, interests: v.interests }),
    onSuccess: (p) => { sync(p); next(); },
    onError,
  });

  const finish = () => void qc.invalidateQueries({ queryKey: keys.profile });

  return (
    <Screen scroll>
      <View style={{ flexDirection: "row", gap: 6 }} accessibilityLabel={`Étape ${step + 1} sur ${STEPS.length}`}>
        {STEPS.map((s, i) => (
          <View key={s} style={{ flex: 1, height: 5, borderRadius: radius.pill, backgroundColor: i <= step ? c.accent : c.border }} />
        ))}
      </View>
      <Text variant="label" tone="accent">Étape {step + 1}/{STEPS.length} — {STEPS[step]}</Text>

      {step === 0 ? (
        <>
          <Text variant="title">Faisons connaissance</Text>
          <BasicsForm profile={profile.firstName ? profile : null} submitLabel="Continuer" busy={saveBasics.isPending} onSubmit={(v) => saveBasics.mutate(v)} />
        </>
      ) : null}
      {step === 1 ? (
        <>
          <Text variant="title">Que recherchez-vous ?</Text>
          <PreferencesForm initial={profile.preferences} submitLabel="Continuer" busy={savePrefs.isPending} onSubmit={(p) => savePrefs.mutate(p)} />
        </>
      ) : null}
      {step === 2 ? (
        <>
          <Text variant="title">Autour de vous</Text>
          <LocationStep hasLocation={profile.hasLocation} initialCity={profile.city} onDone={next} />
        </>
      ) : null}
      {step === 3 ? (
        <>
          <Text variant="title">Parlez-nous de vous</Text>
          <AboutForm profile={profile} submitLabel="Continuer" busy={saveAbout.isPending} onSubmit={(v) => saveAbout.mutate(v)} />
        </>
      ) : null}
      {step === 4 ? (
        <>
          <Text variant="title">Vos photos</Text>
          <Text tone="muted">Ajoutez au moins une photo. La première sera votre photo principale.</Text>
          <PhotosManager profile={profile} />
          <Button title="Terminer et découvrir" onPress={finish} disabled={profile.photos.length === 0} testID="onboarding-finish" />
        </>
      ) : null}

      <View style={{ flexDirection: "row", justifyContent: "space-between", marginTop: spacing.md }}>
        {step > 0 ? <Button title="Retour" variant="ghost" onPress={() => setStep((s) => s - 1)} /> : <View />}
        <Button title="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
      </View>
    </Screen>
  );
}
