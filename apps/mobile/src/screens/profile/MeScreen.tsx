import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMyProfile } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Photo, Row, Screen, Section, Text } from "../../ui";
import { interestedInLabels } from "../../utils/format";

export function MeScreen() {
  const { radii } = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { data } = useMyProfile();
  if (!data?.profile) {
    return null;
  }
  const p = data.profile;
  const prefs = data.preferences;

  return (
    <Screen testID="me-screen">
      <Text variant="title">Mon profil</Text>
      <View style={styles.hero}>
        <Photo
          url={data.photos[0]?.url}
          rounded={radii.xl}
          style={styles.photo}
          accessibilityLabel="Ta photo principale"
        />
        <Text variant="title">
          {p.firstName}, {p.age}
        </Text>
        {!data.discoverable ? (
          <Text tone="primary" variant="label">
            Profil masqué de la découverte
          </Text>
        ) : null}
      </View>
      <Section title="Profil">
        <Row
          icon="person-outline"
          label="Mes informations"
          onPress={() => navigation.navigate("EditProfile")}
          testID="me-edit-profile"
        />
        <Row
          icon="images-outline"
          label="Mes photos"
          value={`${data.photos.length}/6`}
          onPress={() => navigation.navigate("EditPhotos")}
          testID="me-edit-photos"
        />
        <Row
          icon="sparkles-outline"
          label="Centres d'intérêt"
          value={`${data.interests.length}`}
          onPress={() => navigation.navigate("EditInterests")}
          last
        />
      </Section>
      <Section title="Rencontres">
        <Row
          icon="options-outline"
          label="Préférences"
          value={prefs ? `${prefs.ageMin}-${prefs.ageMax} ans · ${prefs.maxDistanceKm} km` : undefined}
          onPress={() => navigation.navigate("EditPreferences")}
          testID="me-edit-preferences"
        />
        <Row
          icon="heart-outline"
          label="Je recherche"
          value={prefs?.interestedIn.map((g) => interestedInLabels[g].replace("Des ", "")).join(", ")}
          last
        />
      </Section>
      <Section title="Paramètres">
        <Row
          icon="shield-checkmark-outline"
          label="Confidentialité et sécurité"
          onPress={() => navigation.navigate("Privacy")}
          testID="me-privacy"
        />
        <Row
          icon="settings-outline"
          label="Compte"
          onPress={() => navigation.navigate("Account")}
          testID="me-account"
          last
        />
      </Section>
      <Text variant="caption" tone="subtle" align="center">
        Seren est gratuit : pas d’abonnement, pas de publicité, pas d’option payante.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: 10 },
  photo: { width: 132, height: 168 }
});
