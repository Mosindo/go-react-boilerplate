import React from "react";
import { View } from "react-native";
import type { Gender } from "../api/types";
import { t } from "../i18n/fr";
import { genderLabel, genderPluralLabel, toggleInList } from "../lib/format";
import { useInterests } from "../queries";
import { createStyles } from "../theme/ThemeProvider";
import { Button, Chip, ErrorState, Skeleton, Text, TextField } from "../ui";
import { useApproxLocation } from "./useApproxLocation";

export const GENDERS: Gender[] = ["man", "woman", "non_binary"];

const useStyles = createStyles(({ spacing, colors }) => ({
  group: { gap: spacing.sm },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  dates: { flexDirection: "row", gap: spacing.md },
  small: { flex: 1 },
  wide: { flex: 1.6 },
  note: { color: colors.textMuted }
}));

export function GenderPicker({
  label,
  value,
  onChange,
  error,
  testPrefix
}: {
  label: string;
  value: Gender | null;
  onChange: (gender: Gender) => void;
  error?: string | null;
  testPrefix: string;
}) {
  const styles = useStyles();
  return (
    <View style={styles.group}>
      <Text variant="label">{label}</Text>
      <View style={styles.wrap}>
        {GENDERS.map((gender) => (
          <Chip
            key={gender}
            label={genderLabel(gender)}
            onPress={() => onChange(gender)}
            selected={value === gender}
            testID={`${testPrefix}-${gender}`}
          />
        ))}
      </View>
      {error ? (
        <Text variant="caption" tone="danger">
          {t.validation.required}
        </Text>
      ) : null}
    </View>
  );
}

export function GenderMultiPicker({
  label,
  value,
  onChange,
  error,
  testPrefix
}: {
  label: string;
  value: Gender[];
  onChange: (genders: Gender[]) => void;
  error?: string | null;
  testPrefix: string;
}) {
  const styles = useStyles();
  return (
    <View style={styles.group}>
      <Text variant="label">{label}</Text>
      <View style={styles.wrap}>
        {GENDERS.map((gender) => (
          <Chip
            key={gender}
            label={genderPluralLabel(gender)}
            onPress={() => onChange(toggleInList(value, gender))}
            selected={value.includes(gender)}
            testID={`${testPrefix}-${gender}`}
          />
        ))}
      </View>
      {error ? (
        <Text variant="caption" tone="danger">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

export function BirthDateFields({
  day,
  month,
  year,
  onChange,
  error
}: {
  day: string;
  month: string;
  year: string;
  onChange: (part: "day" | "month" | "year", value: string) => void;
  error?: string | null;
}) {
  const styles = useStyles();
  const digits = (value: string, max: number) => value.replace(/\D/g, "").slice(0, max);
  return (
    <View style={styles.group}>
      <Text variant="label">{t.onboarding.birthDate}</Text>
      <View style={styles.dates}>
        <View style={styles.small}>
          <TextField
            keyboardType="number-pad"
            label={t.onboarding.day}
            maxLength={2}
            onChangeText={(v) => onChange("day", digits(v, 2))}
            testID="onboarding-birth-day"
            value={day}
          />
        </View>
        <View style={styles.small}>
          <TextField
            keyboardType="number-pad"
            label={t.onboarding.month}
            maxLength={2}
            onChangeText={(v) => onChange("month", digits(v, 2))}
            testID="onboarding-birth-month"
            value={month}
          />
        </View>
        <View style={styles.wide}>
          <TextField
            keyboardType="number-pad"
            label={t.onboarding.year}
            maxLength={4}
            onChangeText={(v) => onChange("year", digits(v, 4))}
            testID="onboarding-birth-year"
            value={year}
          />
        </View>
      </View>
      {error ? (
        <Text accessibilityLiveRegion="polite" variant="caption" tone="danger">
          {error}
        </Text>
      ) : null}
    </View>
  );
}

export function InterestPicker({
  value,
  onChange
}: {
  value: string[];
  onChange: (slugs: string[]) => void;
}) {
  const styles = useStyles();
  const interests = useInterests();
  if (interests.isPending) {
    return (
      <View style={styles.wrap}>
        {[80, 110, 90, 70, 100, 120].map((width, index) => (
          <Skeleton height={44} key={index} radius={22} width={width} />
        ))}
      </View>
    );
  }
  if (interests.isError) {
    return <ErrorState onRetry={() => interests.refetch()} title={t.errors.generic} />;
  }
  return (
    <View style={styles.group}>
      <View style={styles.wrap}>
        {interests.data.map((interest) => (
          <Chip
            key={interest.slug}
            label={interest.label}
            onPress={() => onChange(toggleInList(value, interest.slug, 10))}
            selected={value.includes(interest.slug)}
            testID={`interest-${interest.slug}`}
          />
        ))}
      </View>
      <Text variant="caption" tone="muted">
        {t.onboarding.interestsCount(value.length)}
      </Text>
    </View>
  );
}

export function LocationField({
  value,
  onChange
}: {
  value: { latitude: number; longitude: number } | null;
  onChange: (value: { latitude: number; longitude: number } | null) => void;
}) {
  const styles = useStyles();
  const { state, request, clear, openSettings } = useApproxLocation(value);

  React.useEffect(() => {
    if (state.status === "granted")
      onChange({ latitude: state.latitude, longitude: state.longitude });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state]);

  return (
    <View style={styles.group}>
      {state.status === "granted" ? (
        <>
          <Text accessibilityLiveRegion="polite" variant="caption" tone="muted">
            {t.onboarding.locationOk}
          </Text>
          <Button
            label={t.onboarding.locationRemove}
            onPress={() => {
              clear();
              onChange(null);
            }}
            variant="ghost"
          />
        </>
      ) : (
        <Button
          label={t.onboarding.useLocation}
          loading={state.status === "loading"}
          onPress={() => void request()}
          testID="onboarding-location-button"
          variant="secondary"
        />
      )}
      {state.status === "denied" ? (
        <>
          <Text accessibilityLiveRegion="polite" variant="caption" tone="danger">
            {t.onboarding.locationDenied}
          </Text>
          {!state.canAskAgain ? (
            <Button label={t.onboarding.locationSettings} onPress={openSettings} variant="ghost" />
          ) : null}
        </>
      ) : null}
      {state.status === "error" ? (
        <Text variant="caption" tone="danger">
          {t.onboarding.locationError}
        </Text>
      ) : null}
    </View>
  );
}
