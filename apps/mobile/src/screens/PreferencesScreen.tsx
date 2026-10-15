import React, { useEffect, useState } from "react";
import { View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { clearLocation, saveLocation, savePreferences } from "../api/platform";
import type { Gender } from "../api/types";
import { OptionCard } from "../components/OptionCard";
import { Stepper } from "../components/Stepper";
import { queryKeys, useMyProfile } from "../hooks/queries";
import { errorMessage } from "../i18n/errors";
import { distanceChoices, interestedInLabels } from "../i18n/fr";
import { readApproximatePosition } from "../lib/location";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView, showError, showSuccess } from "../shared/feedback";
import { Header, ScreenContainer, Section } from "../shared/layout";
import { Button, Chip, Notice, Text, spacing } from "../shared/ui";

const GENDERS: Gender[] = ["woman", "man", "nonbinary"];

export default function PreferencesScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const query = useMyProfile();
  const profile = query.data?.profile;

  const [seeks, setSeeks] = useState<Gender[]>([]);
  const [minAge, setMinAge] = useState(22);
  const [maxAge, setMaxAge] = useState(40);
  const [distance, setDistance] = useState(50);
  const [ready, setReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [locating, setLocating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (profile && !ready) {
      if (profile.preferences) {
        setSeeks(profile.preferences.interestedIn);
        setMinAge(profile.preferences.minAge);
        setMaxAge(profile.preferences.maxAge);
        setDistance(profile.preferences.maxDistanceKm);
      }
      setReady(true);
    }
  }, [profile, ready]);

  if (query.isError) {
    return (
      <ScreenContainer>
        <ErrorView error={query.error} onRetry={() => void query.refetch()} />
      </ScreenContainer>
    );
  }
  if (!profile || !ready) {
    return (
      <ScreenContainer>
        <LoadingView fullScreen />
      </ScreenContainer>
    );
  }

  const toggle = (value: Gender) => setSeeks((c) => (c.includes(value) ? c.filter((g) => g !== value) : [...c, value]));

  const save = async () => {
    setError(null);
    if (seeks.length === 0) {
      setError("Choisissez au moins une option.");
      return;
    }
    setBusy(true);
    try {
      queryClient.setQueryData(
        queryKeys.myProfile,
        await savePreferences({ interestedIn: seeks, minAge, maxAge, maxDistanceKm: distance })
      );
      showSuccess("Préférences enregistrées.");
      navigation.goBack();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const updateLocation = async () => {
    setLocating(true);
    try {
      const position = await readApproximatePosition();
      if (position.status === "denied") {
        showError("Autorisez la localisation dans les réglages de votre téléphone.");
      } else if (position.status === "unavailable") {
        showError("Position indisponible pour le moment.");
      } else {
        queryClient.setQueryData(queryKeys.myProfile, await saveLocation(position.latitude, position.longitude, position.city));
        showSuccess("Position mise à jour.");
      }
    } catch (e) {
      showError(errorMessage(e));
    } finally {
      setLocating(false);
    }
  };

  const removeLocation = async () => {
    setLocating(true);
    try {
      queryClient.setQueryData(queryKeys.myProfile, await clearLocation());
      showSuccess("Position supprimée.");
    } catch (e) {
      showError(errorMessage(e));
    } finally {
      setLocating(false);
    }
  };

  return (
    <ScreenContainer scroll testID="preferences-screen">
      <Header onBack={() => navigation.goBack()} title="Mes préférences" />

      <Section title="JE SOUHAITE RENCONTRER">
        {GENDERS.map((value) => (
          <OptionCard
            key={value}
            label={interestedInLabels[value]}
            mode="check"
            onPress={() => toggle(value)}
            selected={seeks.includes(value)}
          />
        ))}
      </Section>

      <Section title="ÂGE">
        <Stepper label="Minimum" max={Math.min(maxAge, 99)} min={18} onChange={setMinAge} suffix=" ans" value={minAge} />
        <Stepper label="Maximum" max={99} min={Math.max(minAge, 18)} onChange={setMaxAge} suffix=" ans" value={maxAge} />
      </Section>

      <Section title="DISTANCE MAXIMALE">
        <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
          {distanceChoices.map((km) => (
            <Chip key={km} label={`${km} km`} onPress={() => setDistance(km)} selected={distance === km} />
          ))}
        </View>
      </Section>

      <Section
        description="Arrondie à environ 1 km. Les autres voient uniquement une distance approximative, jamais votre position."
        title="MA POSITION"
      >
        <Text>
          {profile.hasLocation ? (profile.city ? `Enregistrée : ${profile.city}` : "Position enregistrée") : "Aucune position enregistrée"}
        </Text>
        <View style={{ flexDirection: "row", gap: spacing.sm, flexWrap: "wrap" }}>
          <Button
            label={profile.hasLocation ? "Actualiser" : "Utiliser ma position"}
            loading={locating}
            onPress={() => void updateLocation()}
            size="sm"
            variant="secondary"
          />
          {profile.hasLocation ? (
            <Button disabled={locating} label="Supprimer" onPress={() => void removeLocation()} size="sm" variant="ghost" />
          ) : null}
        </View>
      </Section>

      {error ? <Notice message={error} tone="danger" /> : null}
      <Button fullWidth label="Enregistrer" loading={busy} onPress={() => void save()} size="lg" testID="preferences-save" />
    </ScreenContainer>
  );
}
