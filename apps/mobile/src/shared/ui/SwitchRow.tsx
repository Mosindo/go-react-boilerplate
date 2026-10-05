import { StyleSheet, Switch, View } from "react-native";
import { spacing, useTheme } from "../../theme";
import { Text } from "./Text";

export function SwitchRow({ label, description, value, onValueChange, disabled }: { label: string; description?: string; value: boolean; onValueChange: (value: boolean) => void; disabled?: boolean }) {
  const { colors } = useTheme();
  return (
    <View style={styles.row}>
      <View style={styles.texts}>
        <Text variant="bodyStrong">{label}</Text>
        {description ? (
          <Text variant="caption" tone="muted">
            {description}
          </Text>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        value={value}
        disabled={disabled}
        onValueChange={onValueChange}
        trackColor={{ true: colors.primary, false: colors.border }}
        thumbColor="#FFFFFF"
      />
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.lg, paddingVertical: spacing.sm },
  texts: { flex: 1, gap: 2 }
});
