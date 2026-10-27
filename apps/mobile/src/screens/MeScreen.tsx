import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { MainStackParamList } from "../navigation/types";
import { useMyProfile } from "../state/queries";
import { radii, usePalette } from "../theme";
import { Avatar } from "../ui/AuthImage";
import { Screen } from "../ui/Screen";
import { ErrorState, Loading } from "../ui/states";
import { Text } from "../ui/Text";
import { pretty } from "../components/ProfileCard";
import { Chip } from "../ui/Chip";

export function MeScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const p = usePalette();
  const query = useMyProfile();
  const profile = query.data;

  if (query.isLoading)
    return (
      <Screen>
        <Loading />
      </Screen>
    );
  if (query.isError || !profile) {
    return (
      <Screen>
        <ErrorState
          message={query.error?.message ?? "Profile unavailable."}
          onRetry={() => void query.refetch()}
        />
      </Screen>
    );
  }

  const items: { label: string; hint: string; to: keyof MainStackParamList; testID: string }[] = [
    {
      label: "Edit profile",
      hint: "Name, bio, location, interests",
      to: "EditProfile",
      testID: "menu-edit-profile"
    },
    { label: "Photos", hint: `${profile.photos.length} of 6`, to: "Photos", testID: "menu-photos" },
    {
      label: "Discovery preferences",
      hint: `${profile.preferences.ageMin}–${profile.preferences.ageMax}, ${profile.preferences.maxDistanceKm} km`,
      to: "Preferences",
      testID: "menu-preferences"
    },
    {
      label: "Privacy & account",
      hint: profile.discoverable ? "Visible in discovery" : "Hidden from discovery",
      to: "Settings",
      testID: "menu-settings"
    }
  ];

  return (
    <Screen scroll>
      <View style={styles.hero}>
        <Avatar size={112} photoId={profile.photos[0]?.id} label="Your main photo" />
        <Text variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        {profile.city ? <Text tone="muted">{profile.city}</Text> : null}
      </View>
      {profile.bio ? <Text style={styles.center}>{profile.bio}</Text> : null}
      <View style={styles.chips}>
        {profile.interests.map((i) => (
          <Chip key={i} label={pretty(i)} />
        ))}
      </View>
      <View style={[styles.menu, { backgroundColor: p.surface, borderColor: p.border }]}>
        {items.map((item, index) => (
          <Pressable
            key={item.label}
            testID={item.testID}
            accessibilityRole="button"
            accessibilityLabel={item.label}
            onPress={() => navigation.navigate(item.to as never)}
            style={[
              styles.menuRow,
              index > 0 && { borderTopColor: p.border, borderTopWidth: StyleSheet.hairlineWidth }
            ]}
          >
            <View style={styles.flex}>
              <Text variant="label">{item.label}</Text>
              <Text variant="caption" tone="muted">
                {item.hint}
              </Text>
            </View>
            <Text tone="muted">›</Text>
          </Pressable>
        ))}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: 6, paddingTop: 12 },
  center: { textAlign: "center" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8, justifyContent: "center" },
  menu: { borderRadius: radii.lg, borderWidth: 1, overflow: "hidden" },
  menuRow: { flexDirection: "row", alignItems: "center", padding: 16 },
  flex: { flex: 1 }
});
