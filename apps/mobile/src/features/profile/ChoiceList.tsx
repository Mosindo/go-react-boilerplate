import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/tokens";
import { Icon, Text } from "../../ui";

type Option<T extends string> = { value: T; label: string };

/** Large, accessible radio/checkbox list used in onboarding and settings. */
export function ChoiceList<T extends string>({
  options,
  selected,
  onToggle,
  multiple,
  testIDPrefix
}: {
  options: Option<T>[];
  selected: T[];
  onToggle: (value: T) => void;
  multiple?: boolean;
  testIDPrefix?: string;
}) {
  const { colors } = useTheme();
  return (
    <View style={styles.list}>
      {options.map((o) => {
        const active = selected.includes(o.value);
        return (
          <Pressable
            key={o.value}
            accessibilityRole={multiple ? "checkbox" : "radio"}
            accessibilityState={{ checked: active }}
            onPress={() => onToggle(o.value)}
            testID={testIDPrefix ? `${testIDPrefix}-${o.value}` : undefined}
            style={[
              styles.item,
              { borderColor: active ? colors.primary : colors.border, backgroundColor: active ? colors.primarySoft : colors.surface }
            ]}
          >
            <Text style={styles.label} weight={active ? "600" : "400"}>
              {o.label}
            </Text>
            <Icon
              name={multiple ? (active ? "checkbox" : "square-outline") : active ? "radio-button-on" : "radio-button-off"}
              color={active ? colors.primary : colors.textSubtle}
            />
          </Pressable>
        );
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  list: { gap: spacing.xs },
  item: {
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1.5,
    borderRadius: radii.md,
    paddingHorizontal: spacing.md,
    minHeight: 54
  },
  label: { flex: 1 }
});
