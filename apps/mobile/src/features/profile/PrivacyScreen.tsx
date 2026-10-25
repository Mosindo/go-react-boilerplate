import React from "react";
import { Alert } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { profileApi, queryKeys, type ProfileInput, type ProfileState } from "../../api";
import { showToast } from "../../shared/feedback";
import { Button, ErrorState, Screen, Skeleton, SwitchRow, Text } from "../../shared/ui";
import { bumpDeckEpoch } from "../discovery/deckEpoch";
import { useProfileState } from "./hooks";
import { detectLocation } from "./location";
import { profileToInput } from "./profileInput";

type FlagPatch = Pick<ProfileInput, "discoverable" | "showDistance">;

export function PrivacyScreen() {
  const queryClient = useQueryClient();
  const query = useProfileState();
  const profile = query.data?.profile;

  const setProfile = (update: (state: ProfileState) => ProfileState) =>
    queryClient.setQueryData<ProfileState>(queryKeys.profile, (state) => (state ? update(state) : state));

  const updateFlags = useMutation({
    mutationFn: (patch: Partial<FlagPatch>) => {
      if (!profile) {
        throw new Error("profile not loaded");
      }
      return profileApi.save(profileToInput(profile, patch));
    },
    onMutate: (patch) => {
      const previous = queryClient.getQueryData<ProfileState>(queryKeys.profile);
      setProfile((state) => (state.profile ? { ...state, profile: { ...state.profile, ...patch } } : state));
      return { previous };
    },
    onSuccess: (updated) => setProfile((state) => ({ ...state, profile: updated })),
    onError: (_error, _patch, context) => {
      if (context?.previous) queryClient.setQueryData(queryKeys.profile, context.previous);
    }
  });

  const updateLocation = useMutation({
    mutationFn: async () => {
      const outcome = await detectLocation();
      if (outcome.status === "granted") {
        return { outcome: outcome.status, profile: await profileApi.setLocation(outcome.location) };
      }
      return { outcome: outcome.status, profile: null };
    },
    onSuccess: ({ outcome, profile: updated }) => {
      if (updated) {
        setProfile((state) => ({ ...state, profile: updated }));
        bumpDeckEpoch();
        showToast("Position mise à jour.");
      } else {
        Alert.alert(
          outcome === "denied" ? "Localisation refusée" : "Position introuvable",
          "Autorisez la localisation dans les réglages de votre téléphone pour utiliser cette option."
        );
      }
    }
  });

  const removeLocation = useMutation({
    mutationFn: profileApi.clearLocation,
    onSuccess: () => {
      setProfile((state) =>
        state.profile ? { ...state, profile: { ...state.profile, hasLocation: false, city: "" } } : state
      );
      bumpDeckEpoch();
      showToast("Position supprimée.");
    }
  });

  if (query.isError) {
    return (
      <Screen>
        <ErrorState onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  if (!profile) {
    return (
      <Screen>
        <Skeleton height={120} />
      </Screen>
    );
  }

  return (
    <Screen scroll testID="privacy-screen">
      <SwitchRow
        description="Désactivez pour mettre votre profil en pause : plus personne ne le verra dans la découverte."
        label="Profil visible"
        onValueChange={(discoverable) => updateFlags.mutate({ discoverable })}
        testID="privacy-discoverable"
        value={profile.discoverable}
      />
      <SwitchRow
        description="Affiche une distance approximative sur votre profil."
        label="Afficher ma distance"
        onValueChange={(showDistance) => updateFlags.mutate({ showDistance })}
        testID="privacy-show-distance"
        value={profile.showDistance}
      />
      <Text variant="heading">Position</Text>
      <Text color="textMuted">
        {profile.hasLocation
          ? `Position enregistrée${profile.city ? ` : ${profile.city}` : ""}. Les coordonnées sont arrondies à environ 1 km.`
          : "Aucune position enregistrée."}
      </Text>
      <Button
        label={profile.hasLocation ? "Mettre à jour ma position" : "Activer la localisation"}
        loading={updateLocation.isPending}
        onPress={() => updateLocation.mutate()}
        variant="secondary"
      />
      {profile.hasLocation ? (
        <Button
          label="Supprimer ma position"
          loading={removeLocation.isPending}
          onPress={() => removeLocation.mutate()}
          variant="ghost"
        />
      ) : null}
    </Screen>
  );
}
