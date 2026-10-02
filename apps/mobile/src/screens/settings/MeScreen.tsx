import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import React from "react";
import { Pressable, StyleSheet, View } from "react-native";

import type { Card } from "../../api/types";
import { useProfileStatus } from "../../hooks";
import type { MainStackParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { PhotoCarousel } from "../../ui/PhotoCarousel";
import { ProfileBody } from "../../ui/ProfileBody";
import { Screen } from "../../ui/Screen";
import { ErrorState, Loading } from "../../ui/States";
import { Text } from "../../ui/Text";

type Nav = NativeStackNavigationProp<MainStackParamList>;

function Row({
  icon,
  label,
  onPress,
  testID,
}: {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  onPress: () => void;
  testID?: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [
        styles.row,
        {
          backgroundColor: pressed ? colors.surfaceAlt : colors.surface,
          borderColor: colors.border,
        },
      ]}
    >
      <Ionicons name={icon} size={22} color={colors.primary} />
      <Text style={styles.flex}>{label}</Text>
      <Ionicons name="chevron-forward" size={18} color={colors.textMuted} />
    </Pressable>
  );
}

export function MeScreen() {
  const navigation = useNavigation<Nav>();
  const status = useProfileStatus();

  if (status.isLoading) return <Loading />;
  if (status.isError || !status.data?.profile)
    return <ErrorState error={status.error} onRetry={() => void status.refetch()} />;

  const p = status.data.profile;
  // Same shape as what others see, so the preview is faithful.
  const preview: Card = {
    id: "me",
    firstName: p.firstName,
    age: p.age,
    gender: p.gender,
    bio: p.bio,
    city: p.city,
    interests: p.interests,
    photos: p.photos,
  };

  return (
    <Screen scroll edges={["top"]}>
      <Text variant="title">Mon profil</Text>
      <PhotoCarousel photos={preview.photos} name={preview.firstName} />
      <ProfileBody card={preview} />
      <View style={styles.list}>
        <Row
          icon="create-outline"
          label="Modifier mes informations"
          onPress={() => navigation.navigate("EditProfile")}
          testID="me-edit"
        />
        <Row
          icon="images-outline"
          label="Mes photos"
          onPress={() => navigation.navigate("Photos")}
        />
        <Row
          icon="options-outline"
          label="Mes préférences de rencontre"
          onPress={() => navigation.navigate("Preferences")}
        />
        <Row
          icon="settings-outline"
          label="Réglages et confidentialité"
          onPress={() => navigation.navigate("Settings")}
          testID="me-settings"
        />
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: { gap: spacing.sm },
  row: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.lg,
    borderRadius: radii.md,
    borderWidth: 1,
  },
  flex: { flex: 1 },
});
