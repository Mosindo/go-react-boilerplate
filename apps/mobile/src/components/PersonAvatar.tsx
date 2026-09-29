import React from "react";
import { Image, View } from "react-native";
import { useTheme } from "../shared/ui/theme";
import type { Photo } from "../api/models";
import { Txt } from "./kit";
import { photoUri } from "./photoUrl";

type Props = {
  name: string;
  photo: Photo | null | undefined;
  size?: number;
  testID?: string;
};

/** Round avatar showing the photo, or initials as a fallback. */
export const PersonAvatar = React.memo(function PersonAvatar({
  name,
  photo,
  size = 52,
  testID
}: Props) {
  const { colors } = useTheme();
  const uri = photoUri(photo);
  return (
    <View
      testID={testID}
      accessibilityElementsHidden
      importantForAccessibility="no-hide-descendants"
      style={{
        width: size,
        height: size,
        borderRadius: size / 2,
        overflow: "hidden",
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: colors.primarySoft
      }}
    >
      {uri ? (
        <Image source={{ uri }} resizeMode="cover" style={{ width: size, height: size }} />
      ) : (
        <Txt variant="subheading" tone="primary" weight="bold">
          {name.trim().charAt(0).toUpperCase() || "?"}
        </Txt>
      )}
    </View>
  );
});
