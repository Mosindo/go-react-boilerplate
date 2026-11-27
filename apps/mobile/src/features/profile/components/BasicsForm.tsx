import React from "react";
import { StyleSheet, View } from "react-native";
import { GENDER_LABELS, type Gender } from "../../../api/types";
import { spacing } from "../../../theme/palette";
import { Chip, Text, TextField } from "../../../ui";

export type BasicsValue = {
  firstName: string;
  day: string;
  month: string;
  year: string;
  gender: Gender | "";
  city: string;
};

type Props = {
  value: BasicsValue;
  onChange: (next: BasicsValue) => void;
  errors?: Partial<Record<"firstName" | "birthdate" | "gender", string | null>>;
};

const digits = (v: string, max: number) => v.replace(/[^0-9]/g, "").slice(0, max);

export function BasicsForm({ value, onChange, errors }: Props) {
  const set = (patch: Partial<BasicsValue>) => onChange({ ...value, ...patch });
  return (
    <View style={styles.wrap}>
      <TextField
        label="Prénom"
        value={value.firstName}
        onChangeText={(firstName) => set({ firstName })}
        maxLength={40}
        autoComplete="given-name"
        textContentType="givenName"
        error={errors?.firstName}
        testID="profile-first-name"
      />
      <View style={styles.dateWrap}>
        <Text variant="label" tone="muted">
          Date de naissance
        </Text>
        <View style={styles.dateRow}>
          <View style={styles.dateSmall}>
            <TextField label="Jour" value={value.day} onChangeText={(d) => set({ day: digits(d, 2) })} keyboardType="number-pad" placeholder="JJ" testID="profile-day" />
          </View>
          <View style={styles.dateSmall}>
            <TextField label="Mois" value={value.month} onChangeText={(m) => set({ month: digits(m, 2) })} keyboardType="number-pad" placeholder="MM" testID="profile-month" />
          </View>
          <View style={styles.dateLarge}>
            <TextField label="Année" value={value.year} onChangeText={(y) => set({ year: digits(y, 4) })} keyboardType="number-pad" placeholder="AAAA" testID="profile-year" />
          </View>
        </View>
        {errors?.birthdate ? (
          <Text variant="caption" tone="danger">
            {errors.birthdate}
          </Text>
        ) : (
          <Text variant="caption" tone="subtle">
            Seul ton âge sera visible.
          </Text>
        )}
      </View>
      <View style={styles.group}>
        <Text variant="label" tone="muted">
          Je suis
        </Text>
        <View style={styles.chips}>
          {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
            <Chip key={g} label={GENDER_LABELS[g]} selected={value.gender === g} onPress={() => set({ gender: g })} testID={`gender-${g}`} />
          ))}
        </View>
        {errors?.gender ? (
          <Text variant="caption" tone="danger">
            {errors.gender}
          </Text>
        ) : null}
      </View>
      <TextField
        label="Ville (facultatif)"
        value={value.city}
        onChangeText={(city) => set({ city })}
        maxLength={80}
        autoComplete="postal-address-locality"
        hint="Affichée sur ton profil, à toi de choisir le niveau de précision."
        testID="profile-city"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: spacing.md },
  dateWrap: { gap: 6 },
  dateRow: { flexDirection: "row", gap: spacing.sm },
  dateSmall: { flex: 1 },
  dateLarge: { flex: 1.4 },
  group: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
