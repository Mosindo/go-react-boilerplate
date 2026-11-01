import { useEffect, useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { GENDER_LABELS, profileApi } from "../api";
import type { Gender } from "../api/types";
import { useToast } from "../components/feedback";
import { InterestPicker } from "../components/InterestPicker";
import { LocationPicker, type Place } from "../components/LocationPicker";
import { Button, Chip, ErrorState, Loading, Screen, Text, TextField } from "../components/ui";
import { formatBirthInput } from "../lib/date";
import { errorMessage } from "../lib/errors";
import type { RootStackParamList } from "../navigation/types";
import { keys, useMyProfile } from "../state/queries";
import { spacing } from "../theme";

type Props = NativeStackScreenProps<RootStackParamList, "EditProfile">;
const GENDERS: Gender[] = ["woman", "man", "nonbinary"];

export function EditProfileScreen({ navigation }: Props) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const profile = useMyProfile();
  const [loaded, setLoaded] = useState(false);
  const [firstName, setFirstName] = useState("");
  const [gender, setGender] = useState<Gender>("woman");
  const [bio, setBio] = useState("");
  const [place, setPlace] = useState<Place | null>(null);
  const [interests, setInterests] = useState<string[]>([]);
  const [busy, setBusy] = useState(false);
  const [nameError, setNameError] = useState<string | null>(null);

  useEffect(() => {
    if (profile.data && !loaded) {
      const p = profile.data;
      setFirstName(p.firstName);
      setGender(p.gender);
      setBio(p.bio);
      setPlace({ city: p.city, latitude: p.latitude, longitude: p.longitude });
      setInterests(p.interests.map((i) => i.slug));
      setLoaded(true);
    }
  }, [profile.data, loaded]);

  if (profile.isLoading) {
    return <Loading />;
  }
  if (profile.isError || !profile.data) {
    return <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />;
  }
  const current = profile.data;

  const save = async () => {
    if (!firstName.trim()) {
      setNameError("Prénom requis");
      return;
    }
    if (!place) {
      return;
    }
    setBusy(true);
    try {
      await profileApi.saveMine({
        firstName: firstName.trim(),
        birthDate: current.birthDate,
        gender,
        bio: bio.trim(),
        city: place.city,
        latitude: place.latitude,
        longitude: place.longitude,
        interests,
      });
      await queryClient.invalidateQueries({ queryKey: keys.profile });
      toast.show("Profil mis à jour", "success");
      navigation.goBack();
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen edges={["bottom"]} scroll>
      <TextField error={nameError} label="Prénom" maxLength={40} onChangeText={setFirstName} value={firstName} />
      <TextField editable={false} label="Date de naissance" value={formatBirthInput(current.birthDate)} />
      <Text muted style={{ marginBottom: spacing.sm }} variant="label">
        Je suis
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", marginBottom: spacing.md }}>
        {GENDERS.map((g) => (
          <Chip key={g} label={GENDER_LABELS[g]} onPress={() => setGender(g)} selected={gender === g} />
        ))}
      </View>
      <TextField
        label="Bio"
        maxLength={500}
        multiline
        onChangeText={setBio}
        style={{ minHeight: 96, textAlignVertical: "top" }}
        value={bio}
      />
      <Text style={{ marginBottom: spacing.sm }} variant="heading">
        Localisation
      </Text>
      <LocationPicker onChange={setPlace} value={place} />
      <Text style={{ marginVertical: spacing.md }} variant="heading">
        Centres d'intérêt
      </Text>
      <InterestPicker onChange={setInterests} selected={interests} />
      <View style={{ marginTop: spacing.xl }}>
        <Button loading={busy} onPress={() => void save()} title="Enregistrer" />
      </View>
    </Screen>
  );
}
