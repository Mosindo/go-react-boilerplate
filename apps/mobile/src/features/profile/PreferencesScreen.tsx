import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { profileApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { Gender, MeProfile } from "../../api/types";
import type { RootStackParamList } from "../../navigation/types";
import { makeStyles } from "../../theme";
import { Button, Card, Chip, Header, IconButton, ListRow, LoadingState, Screen, Section, showToast, Text } from "../../ui";
import { seekingLabels } from "../../utils/format";
import { useMyProfile, useSetMyProfile } from "./hooks";
import { shareApproximateLocation } from "./location";

type Props = NativeStackScreenProps<RootStackParamList, "Preferences">;
const genders: Gender[] = ["woman", "man", "nonbinary"];
const distances = [5, 10, 25, 50, 100, 200, 500];

export function PreferencesScreen({ navigation }: Props) {
  const me = useMyProfile().data;
  if (!me?.preferences) return <LoadingState />;
  return <PreferencesForm me={me} onDone={() => navigation.goBack()} />;
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
  const styles = useStyles();
  return (
    <View style={styles.stepper}>
      <Text variant="label" tone="muted">
        {label}
      </Text>
      <View style={styles.stepperRow}>
        <IconButton
          icon="remove"
          accessibilityLabel={`Diminuer ${label}`}
          disabled={value <= min}
          onPress={() => onChange(Math.max(min, value - 1))}
          style={styles.stepButton}
        />
        <Text variant="title" accessibilityLiveRegion="polite" style={styles.stepValue}>
          {value}
        </Text>
        <IconButton
          icon="add"
          accessibilityLabel={`Augmenter ${label}`}
          disabled={value >= max}
          onPress={() => onChange(Math.min(max, value + 1))}
          style={styles.stepButton}
        />
      </View>
    </View>
  );
}

function PreferencesForm({ me, onDone }: { me: MeProfile; onDone: () => void }) {
  const styles = useStyles();
  const queryClient = useQueryClient();
  const setProfile = useSetMyProfile();
  const prefs = me.preferences!;
  const [interestedIn, setInterestedIn] = useState<Gender[]>(prefs.interestedIn);
  const [minAge, setMinAge] = useState(prefs.minAge);
  const [maxAge, setMaxAge] = useState(prefs.maxAge);
  const [distance, setDistance] = useState(prefs.maxDistanceKm);
  const [saving, setSaving] = useState(false);
  const [locating, setLocating] = useState(false);

  const save = async () => {
    if (interestedIn.length === 0) {
      showToast("Choisissez au moins un genre.", "error");
      return;
    }
    setSaving(true);
    try {
      setProfile(await profileApi.preferences({ interestedIn, minAge, maxAge, maxDistanceKm: distance }));
      showToast("Préférences enregistrées.", "success");
      onDone();
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setSaving(false);
    }
  };

  const updateLocation = async () => {
    setLocating(true);
    try {
      const res = await shareApproximateLocation();
      if (res.ok) {
        showToast("Position approximative mise à jour.", "success");
        void queryClient.invalidateQueries({ queryKey: queryKeys.myProfile });
      } else {
        showToast(
          res.reason === "denied" ? "Autorisez la localisation dans les réglages de l'appareil." : "Position indisponible.",
          "error"
        );
      }
    } catch (err) {
      showToast(errorMessage(err), "error");
    } finally {
      setLocating(false);
    }
  };

  const clearLocation = async () => {
    try {
      await profileApi.clearLocation();
      showToast("Position supprimée.", "success");
      void queryClient.invalidateQueries({ queryKey: queryKeys.myProfile });
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  return (
    <Screen scroll footer={<Button testID="preferences-save" label="Enregistrer" size="lg" loading={saving} onPress={() => void save()} />}>
      <Header title="Préférences" />
      <Section title="Je souhaite rencontrer">
        <View style={styles.chips}>
          {genders.map((g) => (
            <Chip
              key={g}
              label={seekingLabels[g]}
              selected={interestedIn.includes(g)}
              onPress={() => setInterestedIn((c) => (c.includes(g) ? c.filter((x) => x !== g) : [...c, g]))}
            />
          ))}
        </View>
      </Section>
      <Section title="Âge">
        <View style={styles.ages}>
          <Stepper label="Minimum" value={minAge} min={18} max={maxAge} onChange={setMinAge} />
          <Stepper label="Maximum" value={maxAge} min={minAge} max={99} onChange={setMaxAge} />
        </View>
      </Section>
      <Section title={`Distance maximale : ${distance} km`} hint="Appliquée uniquement si vous avez partagé votre position.">
        <View style={styles.chips}>
          {distances.map((d) => (
            <Chip key={d} label={`${d} km`} selected={distance === d} onPress={() => setDistance(d)} />
          ))}
        </View>
      </Section>
      <Section
        title="Ma position"
        hint="Arrondie à environ 1 km et jamais montrée aux autres : seule une distance approximative est affichée."
      >
        <Card>
          <ListRow
            icon="navigate-outline"
            label={locating ? "Localisation…" : "Mettre à jour ma position"}
            description={me.profile?.hasLocation ? "Position enregistrée" : "Aucune position enregistrée"}
            onPress={() => void updateLocation()}
          />
          {me.profile?.hasLocation ? (
            <ListRow
              icon="close-circle-outline"
              label="Supprimer ma position"
              tone="danger"
              onPress={() => void clearLocation()}
              divider={false}
            />
          ) : null}
        </Card>
      </Section>
    </Screen>
  );
}

const useStyles = makeStyles((t) => ({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: t.spacing.sm },
  ages: { flexDirection: "row", gap: t.spacing.md },
  stepper: {
    flex: 1,
    backgroundColor: t.colors.surface,
    borderRadius: t.radii.lg,
    borderWidth: 1,
    borderColor: t.colors.border,
    padding: t.spacing.md,
    gap: t.spacing.xs,
    alignItems: "center"
  },
  stepperRow: { flexDirection: "row", alignItems: "center", gap: t.spacing.sm },
  stepButton: { backgroundColor: t.colors.surfaceAlt },
  stepValue: { minWidth: 44, textAlign: "center" }
}));
