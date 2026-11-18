import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/errors";
import type { Gender, Preferences } from "../../api/types";
import { ChoiceGroup } from "../../components/ChoiceGroup";
import { useFeedback } from "../../feedback";
import { useClearLocation, useMyProfile, useSaveLocation, useSavePreferences } from "../../hooks/queries";
import type { AppScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Button, Card, Chip, Header, LoadingState, Screen, Stepper, Text } from "../../ui";
import { distanceOptions, genders, interestedInLabels } from "../../utils/labels";
import { getApproximateLocation } from "../../utils/location";

export default function PreferencesScreen({ navigation }: AppScreenProps<"Preferences">) {
  const query = useMyProfile();
  if (!query.data?.preferences || !query.data.profile) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  return (
    <PreferencesForm
      city={query.data.profile.city}
      hasLocation={query.data.profile.hasLocation}
      initial={query.data.preferences}
      onDone={() => navigation.goBack()}
    />
  );
}

function PreferencesForm({
  initial,
  hasLocation,
  city,
  onDone
}: {
  initial: Preferences;
  hasLocation: boolean;
  city: string;
  onDone: () => void;
}) {
  const { toast, confirm } = useFeedback();
  const save = useSavePreferences();
  const saveLocation = useSaveLocation();
  const clearLocation = useClearLocation();
  const [interestedIn, setInterestedIn] = useState<Gender[]>(initial.interestedIn);
  const [minAge, setMinAge] = useState(initial.minAge);
  const [maxAge, setMaxAge] = useState(initial.maxAge);
  const [distance, setDistance] = useState(initial.maxDistanceKm);
  const [locating, setLocating] = useState(false);

  const submit = async () => {
    if (interestedIn.length === 0) {
      toast("Choisissez au moins une option.", "error");
      return;
    }
    try {
      await save.mutateAsync({ interestedIn, minAge, maxAge, maxDistanceKm: distance });
      toast("Critères enregistrés", "success");
      onDone();
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  const locate = async () => {
    setLocating(true);
    const result = await getApproximateLocation();
    setLocating(false);
    if (result.status !== "ok") {
      toast(result.status === "denied" ? "Autorisez la localisation dans les réglages." : "Position indisponible.", "error");
      return;
    }
    try {
      await saveLocation.mutateAsync(result);
      toast("Position mise à jour", "success");
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  const forget = async () => {
    const ok = await confirm({
      title: "Supprimer ma position ?",
      message: "Les distances ne seront plus affichées et la découverte ne filtrera plus par distance.",
      confirmLabel: "Supprimer",
      destructive: true
    });
    if (ok) {
      try {
        await clearLocation.mutateAsync(undefined);
        toast("Position supprimée", "success");
      } catch (e) {
        toast(errorMessage(e), "error");
      }
    }
  };

  return (
    <Screen scroll testID="preferences-screen">
      <Header onBack={onDone} title="Critères" />
      <ChoiceGroup
        label="Je souhaite rencontrer"
        labels={interestedInLabels}
        multiple
        onChange={setInterestedIn}
        options={genders}
        value={interestedIn}
      />
      <Card>
        <Text variant="bodyStrong">Âge</Text>
        <Stepper label="À partir de" max={maxAge} min={18} onChange={setMinAge} suffix=" ans" value={minAge} />
        <Stepper label="Jusqu’à" max={99} min={minAge} onChange={setMaxAge} suffix=" ans" value={maxAge} />
      </Card>
      <Card>
        <Text variant="bodyStrong">Distance maximale</Text>
        <View style={styles.chips}>
          {distanceOptions.map((d) => (
            <Chip key={d} label={`${d} km`} onPress={() => setDistance(d)} selected={distance === d} testID={`distance-${d}`} />
          ))}
        </View>
        <Text color="textMuted" variant="caption">
          {hasLocation
            ? `Position enregistrée${city ? ` (${city})` : ""}, arrondie à environ 1 km.`
            : "Aucune position enregistrée : la distance n’est pas prise en compte."}
        </Text>
        <View style={styles.row}>
          <Button
            fullWidth={false}
            icon="navigate-outline"
            label={hasLocation ? "Actualiser" : "Utiliser ma position"}
            loading={locating}
            onPress={locate}
            size="sm"
            variant="secondary"
          />
          {hasLocation ? <Button fullWidth={false} label="Supprimer" onPress={forget} size="sm" variant="ghost" /> : null}
        </View>
      </Card>
      <Button label="Enregistrer" loading={save.isPending} onPress={submit} testID="preferences-save" />
    </Screen>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  row: { flexDirection: "row", gap: spacing.xs, flexWrap: "wrap" }
});
