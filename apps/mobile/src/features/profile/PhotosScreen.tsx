import React from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import type { AppStackParamList } from "../../navigation/types";
import { useSession } from "../../state/session";
import { Header, LoadingState, Screen, Text } from "../../ui";
import { PhotoGrid } from "./PhotoGrid";

type Props = NativeStackScreenProps<AppStackParamList, "Photos">;

export function PhotosScreen({ navigation }: Props) {
  const { profile } = useSession();
  return (
    <Screen scroll>
      <Header title="Mes photos" onBack={navigation.goBack} />
      <Text tone="muted" style={{ marginBottom: 16 }}>
        Touchez une photo pour la définir comme principale, la déplacer, la remplacer ou la supprimer.
      </Text>
      {profile ? <PhotoGrid profile={profile} /> : <LoadingState />}
    </Screen>
  );
}
