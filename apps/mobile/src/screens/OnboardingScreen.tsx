import React from "react";
import { View } from "react-native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { profileApi } from "../api/endpoints";
import type { MissingStep, Overview } from "../api/types";
import { useAuth } from "../auth/AuthContext";
import { PhotosEditor } from "../features/PhotosEditor";
import { PreferencesForm } from "../features/PreferencesForm";
import { ProfileForm } from "../features/ProfileForm";
import { queryKeys } from "../lib/queryKeys";
import { Button, Screen, Text } from "../ui";

const ORDER: MissingStep[] = ["profile", "preferences", "photos"];

/**
 * Onboarding is driven by the server's `missing` list, so it resumes at the
 * right step after an app restart and cannot be bypassed from the client.
 */
export function OnboardingScreen({
  overview,
  onFinish
}: {
  overview: Overview;
  onFinish: () => void;
}) {
  const qc = useQueryClient();
  const { signOut } = useAuth();
  const step = ORDER.find((s) => overview.missing.includes(s)) ?? "photos";
  const index = ORDER.indexOf(step) + 1;
  const refetch = () => qc.invalidateQueries({ queryKey: queryKeys.overview });

  return (
    <Screen scroll edges={["top", "bottom"]}>
      <Text variant="caption" tone="muted">
        Étape {index} sur 3
      </Text>
      {step === "profile" ? (
        <>
          <Text variant="title">Faisons connaissance</Text>
          <ProfileForm
            initial={overview.profile}
            submitLabel="Continuer"
            onSaved={() => void refetch()}
          />
        </>
      ) : null}
      {step === "preferences" ? (
        <>
          <Text variant="title">Ce que vous cherchez</Text>
          <PreferencesForm
            initial={overview.preferences}
            submitLabel="Continuer"
            onSaved={() => void refetch()}
          />
        </>
      ) : null}
      {step === "photos" ? <PhotosStep overview={overview} onFinish={onFinish} /> : null}
      <View>
        <Button label="Se déconnecter" variant="ghost" onPress={() => void signOut()} />
      </View>
    </Screen>
  );
}

function PhotosStep({ overview, onFinish }: { overview: Overview; onFinish: () => void }) {
  const live = useQuery({
    queryKey: queryKeys.overview,
    queryFn: profileApi.overview,
    initialData: overview
  });
  const photos = live.data.profile?.photos ?? [];
  return (
    <>
      <Text variant="title">Vos plus belles photos</Text>
      <Text tone="muted">Ajoutez au moins une photo pour apparaître dans les découvertes.</Text>
      <PhotosEditor photos={photos} />
      <Button label="Terminer" disabled={photos.length === 0} onPress={onFinish} />
    </>
  );
}
