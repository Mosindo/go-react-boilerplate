import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useState } from "react";
import { View } from "react-native";
import { getErrorMessage } from "../../api/client";
import type { Gender, OwnProfile, ProfileInput } from "../../api/types";
import { useClearLocation, useOwnProfile, useSaveProfile, useSetLocation } from "../../hooks/useProfile";
import { isoToBirthInput } from "../../lib/dates";
import type { Coordinates } from "../../lib/location";
import { BIO_MAX, validateCity, validateDisplayName } from "../../lib/validation";
import type { MainStackParamList } from "../../navigation/types";
import { Button, ErrorState, Input, LoadingState, Screen, Text, toast } from "../../ui";
import { GenderPicker } from "./GenderPicker";
import { InterestPicker } from "./InterestPicker";
import { LocationField } from "./LocationField";

type Props = NativeStackScreenProps<MainStackParamList, "EditProfile">;

export function EditProfileScreen(props: Props) {
  const query = useOwnProfile();
  if (query.isPending) return <LoadingState />;
  if (query.isError || !query.data) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;
  return <EditForm {...props} profile={query.data} />;
}

function EditForm({ navigation, profile }: Props & { profile: OwnProfile }) {
  const save = useSaveProfile();
  const setLocation = useSetLocation();
  const clearLocation = useClearLocation();
  const [displayName, setDisplayName] = useState(profile.displayName);
  const [gender, setGender] = useState<Gender>(profile.gender);
  const [bio, setBio] = useState(profile.bio);
  const [city, setCity] = useState(profile.city);
  const [interests, setInterests] = useState(profile.interests.map((i) => i.slug));
  const [coordinates, setCoordinates] = useState<Coordinates | null>(null);
  const [errors, setErrors] = useState<{ displayName?: string; city?: string; form?: string }>({});

  async function submit() {
    const next = { displayName: validateDisplayName(displayName) ?? undefined, city: validateCity(city) ?? undefined };
    if (next.displayName || next.city) {
      setErrors(next);
      return;
    }
    setErrors({});
    const input: ProfileInput = {
      displayName: displayName.trim(),
      birthDate: profile.birthDate,
      gender,
      bio: bio.trim(),
      city: city.trim(),
      interests,
      showDistance: profile.showDistance,
      discoverable: profile.discoverable
    };
    try {
      await save.mutateAsync(input);
      if (coordinates) await setLocation.mutateAsync(coordinates);
      toast.success("Profile saved");
      navigation.goBack();
    } catch (e) {
      setErrors({ form: getErrorMessage(e) });
    }
  }

  return (
    <Screen keyboardAvoiding scroll>
      <Input
        autoCapitalize="words"
        error={errors.displayName}
        label="First name"
        maxLength={40}
        onChangeText={setDisplayName}
        value={displayName}
      />
      <Input
        editable={false}
        hint="Your birth date can't be changed here."
        label="Birth date"
        value={isoToBirthInput(profile.birthDate)}
      />
      <GenderPicker onChange={setGender} value={gender} />
      <Input
        hint={`${bio.length}/${BIO_MAX}`}
        label="Bio"
        maxLength={BIO_MAX}
        multiline
        onChangeText={setBio}
        value={bio}
      />
      <LocationField
        city={city}
        coordinates={coordinates}
        error={errors.city}
        hasSavedLocation={profile.hasLocation}
        onCityChange={setCity}
        onCoordinates={setCoordinates}
      />
      {profile.hasLocation ? (
        <Button
          label="Stop sharing my location"
          loading={clearLocation.isPending}
          onPress={() =>
            clearLocation.mutate(undefined, {
              onSuccess: () => toast.success("Location removed"),
              onError: (e) => toast.error(getErrorMessage(e))
            })
          }
          variant="ghost"
        />
      ) : null}
      <Text weight="semibold">Interests</Text>
      <InterestPicker onChange={setInterests} selected={interests} />
      {errors.form ? (
        <Text accessibilityRole="alert" tone="danger">
          {errors.form}
        </Text>
      ) : null}
      <View>
        <Button label="Save changes" loading={save.isPending} onPress={submit} />
      </View>
    </Screen>
  );
}
