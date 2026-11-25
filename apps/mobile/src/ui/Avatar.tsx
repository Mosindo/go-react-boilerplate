import React from "react";
import { StyleSheet, View } from "react-native";
import { colors } from "../theme";
import { RemoteImage } from "./RemoteImage";
import { Text } from "./Text";

type AvatarProps = {
  path?: string | null;
  name: string;
  size?: number;
  ring?: boolean;
};

export function Avatar({ path, name, size = 52, ring }: AvatarProps) {
  const radius = size / 2;
  return (
    <View style={[ring && styles.ring, { borderRadius: radius + 3 }]}>
      {path ? (
        <RemoteImage path={path} style={{ width: size, height: size, borderRadius: radius }} accessibilityLabel={`Photo de ${name}`} />
      ) : (
        <View style={[styles.fallback, { width: size, height: size, borderRadius: radius }]}>
          <Text variant="heading" color="primary">
            {name.slice(0, 1).toUpperCase()}
          </Text>
        </View>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  ring: {
    padding: 2,
    borderWidth: 2,
    borderColor: colors.primary
  },
  fallback: {
    backgroundColor: colors.primarySoft,
    alignItems: "center",
    justifyContent: "center"
  }
});
