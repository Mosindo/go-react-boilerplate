import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Gender, MyProfile } from "../../api/types";
import { useMyProfile } from "../../hooks";
import type { RootScreenProps } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme";
import { Button, Chip, IconButton, LoadingState, Screen, Text, showToast } from "../../ui";
import { InterestedInPicker, LocationPicker } from "./fields";

const DISTANCES = [5, 10, 25, 50, 100, 200, 500];

export function PreferencesScreen(props: RootScreenProps<"Preferences">) {
  const profile = useMyProfile();
  if (!profile.data)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  return <PreferencesForm {...props} profile={profile.data} />;
}

function PreferencesForm({ navigation, profile }: RootScreenProps<"Preferences"> & { profile: MyProfile }) {
  const queryClient = useQueryClient();
  const [interestedIn, setInterestedIn] = useState<Gender[]>(profile.preferences.interestedIn);
  const [ageMin, setAgeMin] = useState(profile.preferences.ageMin);
  const [ageMax, setAgeMax] = useState(profile.preferences.ageMax);
  const [distance, setDistance] = useState(profile.preferences.maxDistanceKm);

  const save = useMutation({
    mutationFn: () => profileApi.updatePreferences({ interestedIn, ageMin, ageMax, maxDistanceKm: distance }),
    onSuccess: (updated) => {
      queryClient.setQueryData(queryKeys.profile, updated);
      void queryClient.resetQueries({ queryKey: queryKeys.discover });
      showToast("Préférences enregistrées.", "success");
      navigation.goBack();
    },
    onError: (e) => showToast(errorMessage(e), "error")
  });

  return (
    <Screen
      scroll
      edges={["left", "right", "bottom"]}
      testID="preferences-screen"
      footer={<Button label="Enregistrer" testID="preferences-save" loading={save.isPending} onPress={() => save.mutate()} />}
    >
      <View style={styles.body}>
        <Text variant="heading">Je souhaite rencontrer</Text>
        <InterestedInPicker value={interestedIn} onChange={setInterestedIn} />

        <Text variant="heading">Âge</Text>
        <View style={styles.ageRow}>
          <Stepper label="Minimum" value={ageMin} min={18} max={ageMax} onChange={setAgeMin} />
          <Stepper label="Maximum" value={ageMax} min={ageMin} max={99} onChange={setAgeMax} />
        </View>

        <Text variant="heading">Distance maximale : {distance} km</Text>
        <View style={styles.wrap}>
          {DISTANCES.map((d) => (
            <Chip key={d} label={`${d} km`} testID={`distance-${d}`} selected={distance === d} onPress={() => setDistance(d)} />
          ))}
        </View>

        <Text variant="heading">Ma zone</Text>
        <LocationPicker currentCity={profile.city} hasLocation={profile.hasLocation} />
      </View>
    </Screen>
  );
}

function Stepper({
  label,
  value,
  min,
  max,
  onChange
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (v: number) => void;
}) {
  return (
    <View style={styles.stepper} accessibilityLabel={`${label} : ${value} ans`}>
      <Text variant="caption" tone="subtle">
        {label}
      </Text>
      <View style={styles.stepperRow}>
        <IconButton
          icon="remove"
          label={`Diminuer l'âge ${label.toLowerCase()}`}
          size={36}
          disabled={value <= min}
          onPress={() => onChange(Math.max(min, value - 1))}
        />
        <Text variant="heading" style={styles.stepperValue}>
          {value}
        </Text>
        <IconButton
          icon="add"
          label={`Augmenter l'âge ${label.toLowerCase()}`}
          size={36}
          disabled={value >= max}
          onPress={() => onChange(Math.min(max, value + 1))}
        />
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  body: {
    gap: spacing.lg,
    paddingTop: spacing.md
  },
  ageRow: {
    flexDirection: "row",
    gap: spacing.md
  },
  stepper: {
    flex: 1,
    gap: spacing.xs,
    padding: spacing.md,
    borderRadius: radii.lg,
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.border
  },
  stepperRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between"
  },
  stepperValue: {
    minWidth: 36,
    textAlign: "center"
  },
  wrap: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  }
});
