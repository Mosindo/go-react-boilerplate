import React from "react";
import { View } from "react-native";
import type { Gender } from "../../api/models";
import { GENDER_INTEREST_LABELS, GENDER_LABELS } from "../../lib/format";
import { toggleInList, GENDERS } from "../../lib/validation";
import { Chip } from "../ui/Chip";
import { FormField } from "../ui/FormField";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

const makeStyles = (t: Theme) => ({
  row: { flexDirection: "row" as const, flexWrap: "wrap" as const, gap: t.spacing.sm }
});

type SingleProps = {
  label: string;
  value: Gender | null;
  onChange: (value: Gender) => void;
  error?: string | null;
  testIDPrefix?: string;
};

export function GenderPicker({
  error,
  label,
  onChange,
  testIDPrefix = "gender",
  value
}: SingleProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <FormField error={error} label={label}>
      <View accessibilityRole="radiogroup" style={styles.row}>
        {GENDERS.map((gender) => (
          <Chip
            accessibilityRole="radio"
            key={gender}
            label={GENDER_LABELS[gender]}
            onPress={() => onChange(gender)}
            selected={value === gender}
            testID={`${testIDPrefix}-${gender}`}
          />
        ))}
      </View>
    </FormField>
  );
}

type MultiProps = {
  label: string;
  value: Gender[];
  onChange: (value: Gender[]) => void;
  error?: string | null;
  testIDPrefix?: string;
};

export function InterestedInPicker({
  error,
  label,
  onChange,
  testIDPrefix = "interested",
  value
}: MultiProps) {
  const styles = useThemedStyles(makeStyles);
  return (
    <FormField error={error} label={label}>
      <View style={styles.row}>
        {GENDERS.map((gender) => (
          <Chip
            key={gender}
            label={GENDER_INTEREST_LABELS[gender]}
            onPress={() => onChange(toggleInList(value, gender))}
            selected={value.includes(gender)}
            testID={`${testIDPrefix}-${gender}`}
          />
        ))}
      </View>
    </FormField>
  );
}
