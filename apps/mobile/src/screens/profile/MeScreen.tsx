import React from "react";
import { StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useAuth } from "../../hooks/useAuth";
import type { AppStackParamList, TabParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { Avatar, Button, Card, Header, ListRow, Screen, Text } from "../../ui";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Me">, NativeStackScreenProps<AppStackParamList>>;

export function MeScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const { profile } = useAuth();
  if (!profile) return null;
  const main = profile.photos[0]?.url;

  return (
    <Screen scroll testID="me-screen">
      <Header large title="Mon profil" />
      <View style={styles.hero}>
        <Avatar name={profile.firstName} ring size={120} url={main} />
        <Text variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        {!profile.discoverable ? (
          <View style={[styles.paused, { backgroundColor: colors.accentSoft }]}>
            <Text tone="accent" variant="label">
              Profil en pause : invisible dans la découverte
            </Text>
          </View>
        ) : null}
        <Button
          icon="eye-outline"
          label="Voir mon profil"
          onPress={() => navigation.navigate("ProfileDetail", { userId: profile.userId, source: "self" })}
          size="md"
          variant="secondary"
        />
      </View>
      <Card>
        <ListRow
          detail="Prénom, bio, métier, centres d'intérêt"
          icon="create-outline"
          label="Modifier le profil"
          onPress={() => navigation.navigate("EditProfile")}
          testID="me-edit"
        />
        <ListRow
          detail={`${profile.photos.length} / 6 photos`}
          icon="images-outline"
          label="Mes photos"
          onPress={() => navigation.navigate("Photos")}
          testID="me-photos"
        />
        <ListRow
          detail="Âge, distance, genres recherchés"
          icon="options-outline"
          label="Préférences de rencontre"
          onPress={() => navigation.navigate("Preferences")}
          testID="me-preferences"
        />
        <ListRow
          detail="Localisation, visibilité, blocages, compte"
          icon="shield-checkmark-outline"
          label="Confidentialité et sécurité"
          onPress={() => navigation.navigate("Settings")}
          testID="me-settings"
        />
      </Card>
      <View style={[styles.free, { backgroundColor: colors.primarySoft, borderRadius: radii.lg }]}>
        <Text tone="primary" variant="bodyStrong">
          Lueur est 100 % gratuit
        </Text>
        <Text tone="muted" variant="caption">
          Likes illimités, voir qui vous a liké, messagerie : tout est inclus, sans abonnement ni publicité.
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.sm, paddingVertical: spacing.lg },
  paused: { paddingHorizontal: spacing.sm, paddingVertical: 4, borderRadius: radii.pill },
  free: { marginTop: spacing.lg, padding: spacing.md, gap: 4 }
});
