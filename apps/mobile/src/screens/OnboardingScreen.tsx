import React, { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import type { OwnProfile, Preferences, ProfileInput } from "../api/types";
import { PhotosEditor } from "../features/PhotosEditor";
import { PreferencesForm } from "../features/PreferencesForm";
import { ProfileForm } from "../features/ProfileForm";
import { keys } from "../features/keys";
import { describeError } from "../lib/validation";
import { Button, Screen, Text, useToast } from "../ui";
import { useAuth } from "../auth/AuthContext";

type Step = "profile" | "photos" | "preferences";
const ORDER: Step[] = ["profile", "photos", "preferences"];

function firstIncomplete(profile: OwnProfile | null): Step {
  if (!profile) return "profile";
  if (profile.photos.length === 0) return "photos";
  return "preferences";
}

export default function OnboardingScreen({ profile }: { profile: OwnProfile | null }) {
  const toast = useToast();
  const { logout } = useAuth();
  const queryClient = useQueryClient();
  const [step, setStep] = useState<Step>(() => firstIncomplete(profile));
  const [current, setCurrent] = useState<OwnProfile | null>(profile);

  const saveProfile = useMutation({
    mutationFn: (input: ProfileInput) => profileApi.save(input),
    onSuccess: (saved) => {
      setCurrent(saved);
      setStep("photos");
    },
    onError: (error) => toast.show(describeError(error), "error")
  });

  const savePrefs = useMutation({
    mutationFn: (prefs: Preferences) => profileApi.savePreferences(prefs),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: keys.profile }),
    onError: (error) => toast.show(describeError(error), "error")
  });

  const index = ORDER.indexOf(step);
  const titles: Record<Step, string> = {
    profile: "Faisons connaissance",
    photos: "Ajoutez vos photos",
    preferences: "Qui souhaitez-vous rencontrer ?"
  };

  return (
    <Screen scroll>
      <Text variant="caption" muted>
        Étape {index + 1} sur {ORDER.length}
      </Text>
      <Text variant="title">{titles[step]}</Text>
      {step === "profile" ? (
        <ProfileForm
          initial={current}
          submitLabel="Continuer"
          busy={saveProfile.isPending}
          onSubmit={(input) => saveProfile.mutate(input)}
        />
      ) : null}
      {step === "photos" ? (
        <>
          <PhotosEditor />
          <PhotosContinue onNext={() => setStep("preferences")} onBack={() => setStep("profile")} />
        </>
      ) : null}
      {step === "preferences" ? (
        <PreferencesForm
          initial={current?.preferences ?? null}
          submitLabel="Terminer"
          busy={savePrefs.isPending}
          onSubmit={(p) => savePrefs.mutate(p)}
        />
      ) : null}
      <Button title="Me déconnecter" variant="ghost" onPress={() => void logout()} />
    </Screen>
  );
}

function PhotosContinue({ onNext, onBack }: { onNext: () => void; onBack: () => void }) {
  const queryClient = useQueryClient();
  const toast = useToast();
  const next = () => {
    const photos = queryClient.getQueryData<{ id: string }[]>(["photos"]) ?? [];
    if (photos.length === 0) {
      toast.show("Ajoutez au moins une photo pour continuer.", "error");
      return;
    }
    onNext();
  };
  return (
    <>
      <Button title="Continuer" onPress={next} testID="photos-continue" />
      <Button title="Retour" variant="ghost" onPress={onBack} />
    </>
  );
}
