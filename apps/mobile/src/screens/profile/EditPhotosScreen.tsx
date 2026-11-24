import React from "react";
import { useNavigation } from "@react-navigation/native";
import { Header } from "../../components/Header";
import { PhotoGrid } from "../../components/profile/PhotoGrid";
import { Screen } from "../../components/Screen";
import { LoadingState } from "../../components/States";
import { useMyProfile } from "../../hooks/queries";

export default function EditPhotosScreen() {
  const navigation = useNavigation();
  const query = useMyProfile();
  return (
    <Screen scroll>
      <Header title="Mes photos" onBack={() => navigation.goBack()} />
      {query.data ? <PhotoGrid photos={query.data.photos} /> : <LoadingState />}
    </Screen>
  );
}
