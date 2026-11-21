import React from "react";
import { View } from "react-native";
import { makeStyles } from "../theme";
import { RemoteImage } from "./RemoteImage";
import { Text } from "./Text";

type AvatarProps = { uri: string | null | undefined; name: string; size?: number; ring?: boolean };

export function Avatar({ uri, name, size = 48, ring }: AvatarProps) {
  const styles = useStyles();
  const dimension = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View style={[styles.base, dimension, ring ? styles.ring : null]} accessibilityLabel={`Photo de ${name}`}>
      {uri ? (
        <RemoteImage uri={uri} style={dimension} />
      ) : (
        <Text variant="heading" tone="muted">
          {name.slice(0, 1).toUpperCase()}
        </Text>
      )}
    </View>
  );
}

const useStyles = makeStyles((t) => ({
  base: { backgroundColor: t.colors.surfaceAlt, alignItems: "center", justifyContent: "center", overflow: "hidden" },
  ring: { borderWidth: 2, borderColor: t.colors.primary }
}));
