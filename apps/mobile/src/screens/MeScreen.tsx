import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import { RemotePhoto } from "../components/RemotePhoto";
import { Row } from "../components/Row";
import { useOwnProfile } from "../hooks/useOwnProfile";
import type { MainStackParamList } from "../navigation/types";
import { Screen } from "../shared/layout";
import { Button, Text, colors, spacing } from "../shared/ui";

export default function MeScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { signOut, user } = useAuth();
  const { data: profile } = useOwnProfile();

  return (
    <Screen scroll>
      <View style={styles.hero}>
        <RemotePhoto style={styles.avatar} url={profile?.photos[0]?.url} />
        <Text variant="heading" weight="bold">
          {profile ? `${profile.firstName}, ${profile.age}` : ""}
        </Text>
        <Text tone="muted">{user?.email}</Text>
      </View>
      <Row
        onPress={() => navigation.navigate("EditProfile")}
        subtitle="Prénom, bio, centres d'intérêt, ville"
        testID="row-edit-profile"
        title="Modifier mon profil"
      />
      <Row onPress={() => navigation.navigate("Photos")} subtitle={`${profile?.photos.length ?? 0}/6 photos`} title="Mes photos" />
      <Row onPress={() => navigation.navigate("Preferences")} subtitle="Genre, âge, distance" title="Préférences de rencontre" />
      <Row
        onPress={() => navigation.navigate("Privacy")}
        subtitle="Visibilité, distance, blocages, compte"
        title="Confidentialité et sécurité"
      />
      <Button label="Se déconnecter" onPress={() => void signOut()} testID="logout" variant="outline" />
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.xs, paddingVertical: spacing.lg },
  avatar: { width: 112, height: 112, borderRadius: 56, overflow: "hidden", borderWidth: 3, borderColor: colors.primaryBorder }
});
