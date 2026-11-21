import React, { useRef, useState } from "react";
import { TextInput, View } from "react-native";
import { makeStyles, useTheme } from "../../theme";
import { Text } from "../../ui";

type Props = { value: string; onChange: (isoDate: string) => void; error?: string | null };

function split(value: string): [string, string, string] {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  return m ? [m[3], m[2], m[1]] : ["", "", ""];
}

/** Day / month / year inputs: simple, keyboard friendly and accessible on every platform. */
export function BirthdateField({ value, onChange, error }: Props) {
  const styles = useStyles();
  const theme = useTheme();
  const [initialDay, initialMonth, initialYear] = split(value);
  const [day, setDay] = useState(initialDay);
  const [month, setMonth] = useState(initialMonth);
  const [year, setYear] = useState(initialYear);
  const monthRef = useRef<TextInput>(null);
  const yearRef = useRef<TextInput>(null);

  const emit = (d: string, m: string, y: string) => {
    if (d.length >= 1 && m.length >= 1 && y.length === 4) {
      onChange(`${y}-${m.padStart(2, "0")}-${d.padStart(2, "0")}`);
    } else {
      onChange("");
    }
  };

  const field = (
    testID: string,
    label: string,
    val: string,
    set: (v: string) => void,
    max: number,
    width: number,
    next?: React.RefObject<TextInput | null>,
    ref?: React.RefObject<TextInput | null>
  ) => (
    <TextInput
      ref={ref}
      testID={testID}
      accessibilityLabel={label}
      placeholder={label}
      placeholderTextColor={theme.colors.textSubtle}
      keyboardType="number-pad"
      maxLength={max}
      value={val}
      onChangeText={(text) => {
        const digits = text.replace(/\D/g, "");
        set(digits);
        if (digits.length === max) next?.current?.focus();
      }}
      style={[styles.input, { width }, error ? styles.errored : null]}
    />
  );

  return (
    <View style={styles.wrapper}>
      <Text variant="label" tone="muted">
        Date de naissance
      </Text>
      <View style={styles.row}>
        {field(
          "birth-day",
          "JJ",
          day,
          (v) => {
            setDay(v);
            emit(v, month, year);
          },
          2,
          72,
          monthRef
        )}
        {field(
          "birth-month",
          "MM",
          month,
          (v) => {
            setMonth(v);
            emit(day, v, year);
          },
          2,
          72,
          yearRef,
          monthRef
        )}
        {field(
          "birth-year",
          "AAAA",
          year,
          (v) => {
            setYear(v);
            emit(day, month, v);
          },
          4,
          104,
          undefined,
          yearRef
        )}
      </View>
      {error ? (
        <Text variant="caption" tone="danger">
          {error}
        </Text>
      ) : (
        <Text variant="caption" tone="subtle">
          Seul votre âge sera visible.
        </Text>
      )}
    </View>
  );
}

const useStyles = makeStyles((t) => ({
  wrapper: { gap: t.spacing.xs },
  row: { flexDirection: "row", gap: t.spacing.sm },
  input: {
    minHeight: 52,
    borderRadius: t.radii.md,
    borderWidth: 1,
    borderColor: t.colors.border,
    backgroundColor: t.colors.surface,
    color: t.colors.text,
    fontSize: 18,
    textAlign: "center"
  },
  errored: { borderColor: t.colors.danger }
}));
