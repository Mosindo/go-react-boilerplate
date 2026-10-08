import React, { useState } from "react";
import { Platform, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { describeError } from "../api/client";
import { keys } from "../api/keys";
import { clearLocation, saveLocation } from "../api/platform";
import type { LookingFor } from "../api/types";
import { ChoiceGroup, InterestPicker, LocationField, type Option } from "../components/Pickers";
import { PhotoManager } from "../components/PhotoManager";
import { useMe, useSaveProfile } from "../hooks/data";
import type { MainStackParamList } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { GENDER_LABELS, LOOKING_FOR_LABELS } from "../shared/format";
import { ScreenContainer, Section } from "../shared/layout";
import { Button, FormField, Notice, Text, spacing } from "../shared/ui";

const LOOKING_OPTIONS: Option<LookingFor>[] = (Object.keys(LOOKING_FOR_LABELS) as LookingFor[]).map((value) => ({ value, label: LOOKING_FOR_LABELS[value] }));

export default function EditProfileScreen({ navigation }: NativeStackScreenProps<MainStackParamList, "EditProfile">) {
  const { data: me, isLoading } = useMe();
  const client = useQueryClient();
  const save = useSaveProfile();
  const profile = me?.profile;

  const [bio, setBio] = useState(profile?.bio ?? "");
  const [occupation, setOccupation] = useState(profile?.occupation ?? "");
  const [city, setCity] = useState(profile?.city ?? "");
  const [lookingFor, setLookingFor] = useState<LookingFor[]>([profile?.lookingFor ?? "open"]);
  const [interests, setInterests] = useState<string[]>(profile?.interests.map((i) => i.slug) ?? []);
  const [coords, setCoords] = useState<{ latitude: number; longitude: number } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [clearedLocation, setClearedLocation] = useState(false);

  if (isLoading) return <LoadingView fullScreen />;
  if (!me || !profile) return <ErrorView message="Profil introuvable." />;

  const submit = async () => {
    setError(null);
    try {
      await save.mutateAsync({
        firstName: profile.firstName,
        birthdate: profile.birthdate,
        gender: profile.gender,
        bio,
        occupation,
        lookingFor: lookingFor[0],
        city,
        interests
      });
      if (coords) client.setQueryData(keys.me, await saveLocation(coords.latitude, coords.longitude, city));
      else if (clearedLocation) client.setQueryData(keys.me, await clearLocation());
      showToast("Profil mis à jour.", "success");
      navigation.goBack();
    } catch (e) {
      setError(describeError(e));
    }
  };

  return (
    <ScreenContainer
      footer={
        <View style={{ width: "100%", maxWidth: 520 }}>
          <Button fullWidth loading={save.isPending} onPress={submit} size="lg">
            Enregistrer
          </Button>
        </View>
      }
    >
      {error ? <Notice message={error} tone="error" /> : null}
      <Section hint="La première photo est affichée en premier. Touchez une photo pour la modifier." title="Photos">
        <PhotoManager photos={me.photos} />
      </Section>

      <Section title="À propos">
        <Text tone="muted">
          {profile.firstName}, {profile.age} ans · {GENDER_LABELS[profile.gender]}
        </Text>
        <FormField counter={500} label="À propos de moi" maxLength={500} multiline onChangeText={setBio} value={bio} />
        <FormField label="Profession" maxLength={80} onChangeText={setOccupation} value={occupation} />
      </Section>

      <Section title="Ce que je recherche">
        <ChoiceGroup onChange={setLookingFor} options={LOOKING_OPTIONS} value={lookingFor} />
      </Section>

      <Section title="Centres d'intérêt">
        <InterestPicker onChange={setInterests} selected={interests} />
      </Section>

      <Section title="Localisation">
        <LocationField
          city={city}
          hasLocation={!clearedLocation && (coords !== null || profile.hasLocation)}
          onCityChange={setCity}
          onClear={() => {
            setCoords(null);
            setClearedLocation(true);
          }}
          onLocated={(location) => {
            setClearedLocation(false);
            setCoords({ latitude: location.latitude, longitude: location.longitude });
            if (location.city) setCity(location.city);
          }}
        />
      </Section>
      {Platform.OS === "web" ? null : <View style={{ height: spacing.lg }} />}
    </ScreenContainer>
  );
}
