import React from "react";
import { StyleSheet, TextInput, View, type TextInputProps } from "react-native";
import { radii, useTheme } from "../theme/theme";
import { Text } from "./Text";

type Props = TextInputProps & { label: string; error?: string | null };

export function Input({ label, error, style, multiline, ...rest }: Props) {
  const { colors } = useTheme();
  return (
    <View style={styles.wrap}>
      <Text variant="label" muted>
        {label}
      </Text>
      <TextInput
        accessibilityLabel={label}
        placeholderTextColor={colors.textMuted}
        multiline={multiline}
        {...rest}
        style={[
          styles.input,
          { backgroundColor: colors.surface, borderColor: error ? colors.danger : colors.border, color: colors.text },
          multiline ? styles.multiline : null,
          style
        ]}
      />
      {error ? (
        <Text variant="caption" color={colors.danger}>
          {error}
        </Text>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { gap: 6 },
  input: { minHeight: 50, borderRadius: radii.md, borderWidth: 1, paddingHorizontal: 14, fontSize: 16 },
  multiline: { minHeight: 100, paddingTop: 12, textAlignVertical: "top" }
});
