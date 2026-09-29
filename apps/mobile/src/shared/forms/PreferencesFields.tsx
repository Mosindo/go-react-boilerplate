import React from "react";
import { View } from "react-native";
import { formatKm } from "../../lib/format";
import {
  LIMITS,
  stepAgeMax,
  stepAgeMin,
  stepDistance,
  type PreferencesErrors,
  type PreferencesValues
} from "../../lib/validation";
import { Text } from "../ui/Text";
import { Stepper } from "../ui/Stepper";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";
import { InterestedInPicker } from "./GenderPicker";

type Props = {
  values: PreferencesValues;
  onChange: (values: PreferencesValues) => void;
  errors?: PreferencesErrors;
};

const makeStyles = (t: Theme) => ({
  group: { gap: t.spacing.xs }
});

export function PreferencesFields({ errors, onChange, values }: Props) {
  const styles = useThemedStyles(makeStyles);
  return (
    <View style={styles.group}>
      <InterestedInPicker
        error={errors?.interestedIn}
        label="I'd like to meet"
        onChange={(interestedIn) => onChange({ ...values, interestedIn })}
        value={values.interestedIn}
      />
      <Stepper
        canDecrement={values.ageMin > LIMITS.ageMin}
        canIncrement={values.ageMin < LIMITS.ageMax}
        label="Youngest"
        onDecrement={() => onChange(stepAgeMin(values, -1))}
        onIncrement={() => onChange(stepAgeMin(values, 1))}
        testID="pref-age-min"
        valueLabel={String(values.ageMin)}
      />
      <Stepper
        canDecrement={values.ageMax > LIMITS.ageMin}
        canIncrement={values.ageMax < LIMITS.ageMax}
        label="Oldest"
        onDecrement={() => onChange(stepAgeMax(values, -1))}
        onIncrement={() => onChange(stepAgeMax(values, 1))}
        testID="pref-age-max"
        valueLabel={String(values.ageMax)}
      />
      {errors?.ageMin || errors?.ageMax ? (
        <Text tone="danger" variant="caption">
          {errors.ageMin ?? errors.ageMax}
        </Text>
      ) : null}
      <Stepper
        canDecrement={values.maxDistanceKm > LIMITS.distanceMin}
        canIncrement={values.maxDistanceKm < LIMITS.distanceMax}
        label="Max distance"
        onDecrement={() =>
          onChange({ ...values, maxDistanceKm: stepDistance(values.maxDistanceKm, -1) })
        }
        onIncrement={() =>
          onChange({ ...values, maxDistanceKm: stepDistance(values.maxDistanceKm, 1) })
        }
        testID="pref-distance"
        valueLabel={formatKm(values.maxDistanceKm)}
      />
      {errors?.maxDistanceKm ? (
        <Text tone="danger" variant="caption">
          {errors.maxDistanceKm}
        </Text>
      ) : null}
    </View>
  );
}
