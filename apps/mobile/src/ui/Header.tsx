import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useTheme } from "../theme";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type Props = {
  title: string;
  subtitle?: string;
  back?: boolean;
  right?: React.ReactNode;
  large?: boolean;
};

export function Header({ title, subtitle, back, right, large }: Props) {
  const navigation = useNavigation();
  const { spacing } = useTheme();
  return (
    <View style={[styles.row, { paddingHorizontal: back ? spacing.xs : spacing.lg, paddingVertical: spacing.xs }]}>
      {back ? <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} /> : null}
      <View style={styles.titles}>
        <Text accessibilityRole="header" numberOfLines={1} variant={large ? "title" : "heading"}>
          {title}
        </Text>
        {subtitle ? (
          <Text numberOfLines={1} tone="muted" variant="caption">
            {subtitle}
          </Text>
        ) : null}
      </View>
      {right ? <View style={styles.right}>{right}</View> : null}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", minHeight: 56, gap: 4 },
  titles: { flex: 1, gap: 2 },
  right: { flexDirection: "row", alignItems: "center" }
});
