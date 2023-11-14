import { useState } from 'react';
import axios from 'axios';

const useFetchSpartanInventory = (gamerInfo, includeArmory = false, setHighlightedItems = null) => {
    const [isLoading, setIsLoading] = useState(true);
  const [spartanInventory, setSpartanInventory] = useState(null);
  const [isFetched, setIsFetched] = useState(false);
  const [armoryRow, setArmoryRow] = useState(null); // State for ArmoryRow data
  const [currentlyEquipped, setCurrentlyEquipped] = useState({
    CurrentlyEquippedCore: null,
    CurrentlyEquippedHelmet: null,
    CurrentlyEquippedVisor: null,
    CurrentlyEquippedGlove: null,
    CurrentlyEquippedCoating: null,
  }); // Added state for CurrentlyEquipped
  const fetchSpartanInventory = async (force = false) => {
    if (isFetched && !force) return;
  
    try {
      const queryParams = includeArmory ? '?includeArmory=true' : '';
      const apiUrl = process.env.REACT_APP_API_URL || 'http://localhost:8080';
      const response = await axios.post(`${apiUrl}/spartan${queryParams}`, gamerInfo);
      console.log(response)
      setSpartanInventory(response.data.PlayerInventory[0]);
      if (includeArmory) {
        setArmoryRow(response.data);
        const equippedData = response.data.CurrentlyEquipped;
        setCurrentlyEquipped({
          CurrentlyEquippedCore: equippedData.CurrentlyEquippedCore,
          CurrentlyEquippedHelmet: equippedData.CurrentlyEquippedHelmet,
          CurrentlyEquippedGlove: equippedData.CurrentlyEquippedGlove,
          CurrentlyEquippedVisor: equippedData.CurrentlyEquippedVisor,
          CurrentlyEquippedCoating: equippedData.CurrentlyEquippedCoating,

        });
  
        // Set initial highlightedId here
        const initialCoreHighlight = response.data.ArmoryRow.find(obj => obj.isHighlighted);
        const initialHelmetHighlight = response.data.ArmoryRowHelmets.find(obj => obj.isHighlighted);
        const initialVisorHighlight = response.data.ArmoryRowVisors.find(obj => obj.isHighlighted);
        const initialGloveHighlight = response.data.ArmoryRowGloves.find(obj => obj.isHighlighted);
        const initialCoatingHighlight = response.data.ArmoryRowCoatings.find(obj => obj.isHighlighted);

        if (initialCoreHighlight) {
          setHighlightedItems(items => ({ ...items, armorcoreId: initialCoreHighlight.id }));
        }
        if (initialHelmetHighlight) {
          setHighlightedItems(items => ({ ...items, armorhelmetId: initialHelmetHighlight.id }));
        }
        if (initialVisorHighlight) {
          setHighlightedItems(items => ({ ...items, armorvisorId: initialVisorHighlight.id }));
        }
        if (initialGloveHighlight) {
          setHighlightedItems(items => ({ ...items, armorgloveId: initialGloveHighlight.id }));
        }
        if (initialCoatingHighlight) {
          setHighlightedItems(items => ({ ...items, armorcoatingId: initialCoatingHighlight.id }));
        }
      }
  
      setIsLoading(false);
      setIsFetched(true);
    } catch (error) {
      console.error("Error fetching Spartan inventory:", error);
      setIsLoading(false);
    }
  };
  

  return { spartanInventory, armoryRow, setArmoryRow, isLoading, fetchSpartanInventory, currentlyEquipped, setCurrentlyEquipped };
};

export default useFetchSpartanInventory;