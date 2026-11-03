import React, { useEffect, useState } from "react";
import { Alert, Platform, Text, View } from "react-native";
import * as Location from "expo-location";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { errorMessage } from "../api/client";
import type { MyProfile } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { AboutFields, BasicsFields, PreferenceFields } from "../features/profile/forms";
import {
  draftFromProfile,
  hasErrors,
  toPreferences,
  toProfileInput,
  validateAbout,
  validateBasics,
  validatePreferences,
  type Draft,
  type Errors
} from "../features/profile/draft";
import { PhotoManager } from "../features/profile/PhotoManager";
import { spacing, useTheme } from "../theme";
import { Body, Button, ErrorText, Screen, Title } from "../ui/components";

const STEPS = ["Qui êtes-vous ?", "Parlez-nous de vous", "Vos préférences", "Vos photos", "Votre position"] as const;

/** Guides a new account through everything the server needs to show it. */
export function OnboardingScreen({ profile, onDone }: { profile: MyProfile | undefined; onDone: () => void }) {
  const theme = useTheme();
  const queryClient = useQueryClient();
  const { signOut } = useAuth();
  const [draft, setDraft] = useState<Draft>(() => draftFromProfile(profile));
  const [step, setStep] = useState(profile ? (profile.preferences ? 3 : 2) : 0);
  const [errors, setErrors] = useState<Errors>({});
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Photos are server state: keep them fresh while the user is on step 4.
    if (step === 3) {
      void queryClient.invalidateQueries({ queryKey: ["profile"] });
    }
  }, [step, queryClient]);

  const save = useMutation({
    mutationFn: async () => {
      if (step <= 1) {
        await api.saveProfile(toProfileInput(draft));
      }
      if (step === 2) {
        await api.savePreferences(toPreferences(draft));
      }
    },
    onSuccess: () => setStep((s) => s + 1),
    onError: (e) => setError(errorMessage(e))
  });

  const photoCount = profile?.photos.length ?? 0;

  const next = () => {
    setError(null);
    let found: Errors = {};
    if (step === 0) {
      found = validateBasics(draft);
    } else if (step === 1) {
      found = { ...validateBasics(draft), ...validateAbout(draft) };
    } else if (step === 2) {
      found = validatePreferences(draft);
    } else if (step === 3 && photoCount === 0) {
      setError("Ajoutez au moins une photo pour continuer.");
      return;
    }
    setErrors(found);
    if (hasErrors(found)) {
      return;
    }
    if (step === 0) {
      setStep(1);
    } else if (step <= 2) {
      save.mutate();
    } else {
      setStep((s) => s + 1);
    }
  };

  const finish = async () => {
    await queryClient.invalidateQueries({ queryKey: ["profile"] });
    onDone();
  };

  const locate = useMutation({
    mutationFn: async () => {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (!permission.granted) {
        throw new Error("Autorisation refusée. Vous pourrez l'activer plus tard dans les réglages.");
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
      await api.setLocation(position.coords.latitude, position.coords.longitude);
    },
    onSuccess: finish,
    onError: (e) => setError(errorMessage(e))
  });

  return (
    <Screen>
      <Text style={{ color: theme.primary, fontWeight: "800" }}>{`Étape ${step + 1} sur ${STEPS.length}`}</Text>
      <Title>{STEPS[step]}</Title>

      {step === 0 ? <BasicsFields draft={draft} onChange={setDraft} errors={errors} /> : null}
      {step === 1 ? <AboutFields draft={draft} onChange={setDraft} errors={errors} /> : null}
      {step === 2 ? <PreferenceFields draft={draft} onChange={setDraft} errors={errors} /> : null}
      {step === 3 ? <PhotoManager photos={profile?.photos ?? []} /> : null}
      {step === 4 ? (
        <View style={{ gap: spacing.md }}>
          <Body>
            Partagez votre position pour voir des profils près de chez vous. Elle est arrondie à environ 1 km : personne ne voit jamais
            votre position exacte, seulement une distance approximative.
          </Body>
          <Body muted>Vous pouvez aussi passer cette étape et la configurer plus tard.</Body>
        </View>
      ) : null}

      <ErrorText message={error} />
      <View style={{ flex: 1 }} />
      {step < 4 ? (
        <Button title="Continuer" onPress={next} loading={save.isPending} testID="onboarding-next" />
      ) : (
        <>
          <Button title="Utiliser ma position" onPress={() => locate.mutate()} loading={locate.isPending} testID="onboarding-locate" />
          <Button title="Passer pour l'instant" variant="secondary" onPress={() => void finish()} testID="onboarding-skip" />
        </>
      )}
      {step > 0 && step < 4 ? <Button title="Retour" variant="ghost" onPress={() => setStep((s) => s - 1)} /> : null}
      {step === 0 ? (
        <Button
          title="Se déconnecter"
          variant="ghost"
          onPress={() => {
            if (Platform.OS === "web") {
              void signOut();
            } else {
              Alert.alert("Se déconnecter ?", undefined, [
                { text: "Annuler", style: "cancel" },
                { text: "Se déconnecter", onPress: () => void signOut() }
              ]);
            }
          }}
        />
      ) : null}
    </Screen>
  );
}
