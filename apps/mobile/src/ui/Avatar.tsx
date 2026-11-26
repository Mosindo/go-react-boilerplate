import React from "react";
import { StyleSheet, View } from "react-native";
import { useTheme } from "../theme";
import { RemoteImage } from "./RemoteImage";
import { Text } from "./Text";

type AvatarProps = {
  url?: string | null;
  name: string;
  size?: number;
  ring?: boolean;
};

export function Avatar({ url, name, size = 48, ring }: AvatarProps) {
  const { colors } = useTheme();
  const style = { width: size, height: size, borderRadius: size / 2 };
  return (
    <View
      accessibilityLabel={`Photo de ${name}`}
      style={[style, styles.wrap, { backgroundColor: colors.accentSoft }, ring ? { borderWidth: 2.5, borderColor: colors.primary } : null]}
    >
      {url ? (
        <RemoteImage style={style} url={url} />
      ) : (
        <Text style={{ fontSize: size * 0.4, lineHeight: size * 0.5 }} tone="accent" variant="heading">
          {name.charAt(0).toUpperCase()}
        </Text>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  wrap: { alignItems: "center", justifyContent: "center", overflow: "hidden" }
});
