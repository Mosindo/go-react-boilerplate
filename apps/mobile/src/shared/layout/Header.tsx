import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { spacing } from "../../theme";
import { Text } from "../ui/Text";

export function Header({ title, subtitle, right }: { title: string; subtitle?: string; right?: ReactNode }) {
  return (
    <View style={styles.row}>
      <View style={styles.text}>
        <Text variant="title">{title}</Text>
        {subtitle ? <Text tone="muted">{subtitle}</Text> : null}
      </View>
      {right}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", gap: spacing.md },
  text: { flex: 1, gap: 2 }
});
