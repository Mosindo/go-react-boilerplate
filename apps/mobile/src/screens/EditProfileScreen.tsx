import React, { useEffect, useState } from "react";
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { saveProfile } from "../api/profile";
import { InterestPicker } from "../components/InterestPicker";
import { PhotoManager } from "../components/PhotoManager";
import { ScreenBar } from "../components/ScreenBar";
import { useProfileQuery } from "../hooks/useProfileData";
import { CITIES } from "../lib/cities";
import { LocationDeniedError, readDeviceLocation } from "../lib/location";
import { MAX_BIO, roundCoordinate } from "../lib/validation";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Chip, FormField, Input, Notice, spacing } from "../shared/ui";

export default function EditProfileScreen() {
  const navigation = useNavigation();
  const queryClient = useQueryClient();
  const query = useProfileQuery();
  const profile = query.data;

  const [firstName, setFirstName] = useState("");
  const [bio, setBio] = useState("");
  const [city, setCity] = useState("");
  const [coords, setCoords] = useState({ latitude: 0, longitude: 0 });
  const [interestIds, setInterestIds] = useState<number[]>([]);
  const [saving, setSaving] = useState(false);
  const [locating, setLocating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (profile) {
      setFirstName(profile.firstName);
      setBio(profile.bio);
      setCity(profile.city);
      setCoords({ latitude: profile.latitude, longitude: profile.longitude });
      setInterestIds(profile.interests.map((interest) => interest.id));
    }
  }, [profile]);

  const locate = async () => {
    setLocating(true);
    setError(null);
    try {
      const found = await readDeviceLocation();
      setCoords({ latitude: found.latitude, longitude: found.longitude });
      if (found.city) {
        setCity(found.city);
      }
    } catch (err) {
      setError(
        err instanceof LocationDeniedError
          ? "Position refusée : choisissez plutôt une ville."
          : "Position indisponible : choisissez une ville."
      );
    } finally {
      setLocating(false);
    }
  };

  const save = async () => {
    if (!profile) {
      return;
    }
    if (!firstName.trim()) {
      setError("Indiquez votre prénom.");
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await saveProfile({
        firstName: firstName.trim(),
        birthdate: profile.birthdate,
        gender: profile.gender,
        bio: bio.trim(),
        city: city.trim(),
        latitude: roundCoordinate(coords.latitude),
        longitude: roundCoordinate(coords.longitude),
        interestIds,
        isDiscoverable: profile.isDiscoverable,
        showDistance: profile.showDistance
      });
      await queryClient.invalidateQueries({ queryKey: ["profile"] });
      await queryClient.invalidateQueries({ queryKey: ["discover"] });
      navigation.goBack();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <ScreenContainer contentMaxWidth={520} testID="edit-profile-screen">
      <ScreenBar title="Modifier mon profil" />
      {query.isLoading ? (
        <LoadingView label="Chargement…" />
      ) : query.isError || !profile ? (
        <ErrorView
          actionLabel="Réessayer"
          message={query.error ? errorMessage(query.error) : "Profil introuvable."}
          onAction={() => void query.refetch()}
        />
      ) : (
        <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.flex}>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <FormField label="Photos">
              <PhotoManager />
            </FormField>
            <Input label="Prénom" maxLength={40} onChangeText={setFirstName} testID="edit-name" value={firstName} />
            <Input
              helperText={`${bio.length} / ${MAX_BIO}`}
              label="Bio"
              maxLength={MAX_BIO}
              multiline
              onChangeText={setBio}
              testID="edit-bio"
              value={bio}
            />
            <View style={styles.location}>
              <Input label="Ville" maxLength={80} onChangeText={setCity} testID="edit-city" value={city} />
              <Button label="Actualiser ma position" loading={locating} onPress={locate} size="sm" variant="secondary" />
              <View style={styles.chips}>
                {CITIES.slice(0, 8).map((item) => (
                  <Chip
                    key={item.name}
                    label={item.name}
                    onPress={() => {
                      setCity(item.name);
                      setCoords({ latitude: item.latitude, longitude: item.longitude });
                    }}
                  />
                ))}
              </View>
            </View>
            <FormField label="Centres d'intérêt">
              <InterestPicker onChange={setInterestIds} selected={interestIds} />
            </FormField>
            {error ? <Notice description={error} title="Enregistrement impossible" tone="danger" /> : null}
            <Button label="Enregistrer" loading={saving} onPress={save} size="lg" testID="edit-save" />
          </ScrollView>
        </KeyboardAvoidingView>
      )}
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { gap: spacing.xl, paddingBottom: spacing.xxl },
  location: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
