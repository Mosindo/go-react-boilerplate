import React, { type ReactNode } from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { IconButton } from "./IconButton";
import { Text } from "./Text";

type HeaderProps = {
  title?: string;
  back?: boolean;
  right?: ReactNode;
  large?: boolean;
};

export function Header({ title, back, right, large }: HeaderProps) {
  const navigation = useNavigation();
  return (
    <View style={[styles.wrap, large && styles.large]}>
      {back ? <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} /> : null}
      {title ? (
        <Text accessibilityRole="header" numberOfLines={1} style={styles.title} variant={large ? "display" : "heading"}>
          {title}
        </Text>
      ) : (
        <View style={styles.title} />
      )}
      {right}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { flexDirection: "row", alignItems: "center", gap: 4, minHeight: 56 },
  large: { minHeight: 72, paddingTop: 8 },
  title: { flex: 1 }
});
