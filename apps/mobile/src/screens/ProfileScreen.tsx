import React from "react";
import { Pressable, ScrollView, StyleSheet, View } from "react-native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Ionicons } from "@expo/vector-icons";
import { PhotoImage } from "../components/PhotoImage";
import { useAuth } from "../hooks/useAuth";
import { formatDistance, genderLabel, nameWithAge } from "../lib/format";
import type { RootStackParamList, TabParamList } from "../navigation/types";
import { SafeAreaLayout } from "../shared/layout";
import { Badge, ListItem, Text, radii, spacing, useTheme } from "../shared/ui";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Me">, NativeStackScreenProps<RootStackParamList>>;

export default function ProfileScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { self, user } = useAuth();
  const profile = self?.profile;
  const primary = self?.photos[0];

  const icon = (name: keyof typeof Ionicons.glyphMap) => (
    <View style={[styles.icon, { backgroundColor: colors.primarySoft }]}>
      <Ionicons color={colors.primary} name={name} size={18} />
    </View>
  );

  return (
    <SafeAreaLayout edges={["top", "left", "right"]}>
      <ScrollView contentContainerStyle={styles.content} testID="profile-screen">
        <View style={styles.hero}>
          <Pressable
            accessibilityLabel="Modifier mes photos"
            accessibilityRole="button"
            onPress={() => navigation.navigate("Photos")}
            style={styles.photo}
          >
            <PhotoImage path={primary?.url} style={StyleSheet.absoluteFill} />
          </Pressable>
          <Text variant="title">{profile ? nameWithAge(profile.firstName, profile.age) : "Mon profil"}</Text>
          <Text tone="muted">
            {[profile ? genderLabel(profile.gender) : null, profile?.city, profile ? formatDistance(undefined) : null]
              .filter(Boolean)
              .join(" · ")}
          </Text>
          {profile?.bio ? (
            <Text style={styles.center}>{profile.bio}</Text>
          ) : (
            <Text style={styles.center} tone="muted">
              Ajoutez une bio pour vous présenter.
            </Text>
          )}
          {self && self.interests.length > 0 ? (
            <View style={styles.chips}>
              {self.interests.map((interest) => (
                <Badge key={interest.id} label={interest.label} />
              ))}
            </View>
          ) : null}
          {profile && !profile.discoverable ? <Badge label="Profil masqué de la découverte" tone="danger" /> : null}
        </View>

        <View>
          <ListItem
            leading={icon("create-outline")}
            onPress={() => navigation.navigate("EditProfile")}
            testID="profile-edit"
            title="Modifier mon profil"
          />
          <ListItem
            leading={icon("images-outline")}
            onPress={() => navigation.navigate("Photos")}
            subtitle={`${self?.photos.length ?? 0}/6 photos`}
            title="Mes photos"
          />
          <ListItem leading={icon("options-outline")} onPress={() => navigation.navigate("Preferences")} title="Préférences de rencontre" />
          <ListItem leading={icon("notifications-outline")} onPress={() => navigation.navigate("Notifications")} title="Notifications" />
          <ListItem
            leading={icon("settings-outline")}
            onPress={() => navigation.navigate("Settings")}
            subtitle={user?.email}
            testID="profile-settings"
            title="Paramètres"
          />
        </View>
      </ScrollView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.xl, gap: spacing.xl },
  hero: { alignItems: "center", gap: spacing.sm },
  photo: { width: 132, height: 132, borderRadius: 66, overflow: "hidden", marginBottom: spacing.sm },
  center: { textAlign: "center" },
  chips: { flexDirection: "row", flexWrap: "wrap", justifyContent: "center", gap: spacing.xs },
  icon: { width: 34, height: 34, borderRadius: radii.sm, alignItems: "center", justifyContent: "center" }
});
