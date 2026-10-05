import { StyleSheet, View } from "react-native";
import { FormField, Input } from "../shared/ui";
import { spacing } from "../theme";

export type BirthDateValue = { day: string; month: string; year: string };

export function BirthDateField({
  value,
  onChange,
  error,
  locked
}: {
  value: BirthDateValue;
  onChange: (value: BirthDateValue) => void;
  error?: string | null;
  locked?: boolean;
}) {
  const digits = (text: string, max: number) => text.replace(/\D/g, "").slice(0, max);
  return (
    <FormField
      label="Date de naissance"
      error={error}
      hint={locked ? "La date de naissance ne peut pas être modifiée." : "Vous devez avoir au moins 18 ans. Seul votre âge sera affiché."}
    >
      <View style={styles.row}>
        <View style={styles.small}>
          <Input
            testID="birth-day"
            label="Jour"
            placeholder="JJ"
            keyboardType="number-pad"
            maxLength={2}
            value={value.day}
            editable={!locked}
            onChangeText={(day) => onChange({ ...value, day: digits(day, 2) })}
          />
        </View>
        <View style={styles.small}>
          <Input
            testID="birth-month"
            label="Mois"
            placeholder="MM"
            keyboardType="number-pad"
            maxLength={2}
            value={value.month}
            editable={!locked}
            onChangeText={(month) => onChange({ ...value, month: digits(month, 2) })}
          />
        </View>
        <View style={styles.large}>
          <Input
            testID="birth-year"
            label="Année"
            placeholder="AAAA"
            keyboardType="number-pad"
            maxLength={4}
            value={value.year}
            editable={!locked}
            onChangeText={(year) => onChange({ ...value, year: digits(year, 4) })}
          />
        </View>
      </View>
    </FormField>
  );
}

const styles = StyleSheet.create({ row: { flexDirection: "row", gap: spacing.sm }, small: { flex: 1 }, large: { flex: 1.6 } });
