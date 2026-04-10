/**
 * Product image mapping.
 * Maps SKU prefixes to themed placeholder images.
 * Each product gets 4+ images for gallery/carousel display.
 * Uses picsum.photos with seed for consistent, beautiful images.
 */

interface ProductImageSet {
  main: string;
  gallery: string[];
}

const imagesByCategory: Record<string, ProductImageSet> = {
  'KB-MECH': {
    main: 'https://picsum.photos/seed/keyboard1/600/400',
    gallery: [
      'https://picsum.photos/seed/keyboard1/600/400',
      'https://picsum.photos/seed/keyboard2/600/400',
      'https://picsum.photos/seed/keyboard3/600/400',
      'https://picsum.photos/seed/keyboard4/600/400',
      'https://picsum.photos/seed/keyboard5/600/400',
    ],
  },
  'MS-WIFI': {
    main: 'https://picsum.photos/seed/mouse1/600/400',
    gallery: [
      'https://picsum.photos/seed/mouse1/600/400',
      'https://picsum.photos/seed/mouse2/600/400',
      'https://picsum.photos/seed/mouse3/600/400',
      'https://picsum.photos/seed/mouse4/600/400',
    ],
  },
  'HUB-USB': {
    main: 'https://picsum.photos/seed/usbhub1/600/400',
    gallery: [
      'https://picsum.photos/seed/usbhub1/600/400',
      'https://picsum.photos/seed/usbhub2/600/400',
      'https://picsum.photos/seed/usbhub3/600/400',
      'https://picsum.photos/seed/usbhub4/600/400',
    ],
  },
  'MON-4K': {
    main: 'https://picsum.photos/seed/monitor1/600/400',
    gallery: [
      'https://picsum.photos/seed/monitor1/600/400',
      'https://picsum.photos/seed/monitor2/600/400',
      'https://picsum.photos/seed/monitor3/600/400',
      'https://picsum.photos/seed/monitor4/600/400',
      'https://picsum.photos/seed/monitor5/600/400',
    ],
  },
  'CAM-HD': {
    main: 'https://picsum.photos/seed/webcam1/600/400',
    gallery: [
      'https://picsum.photos/seed/webcam1/600/400',
      'https://picsum.photos/seed/webcam2/600/400',
      'https://picsum.photos/seed/webcam3/600/400',
      'https://picsum.photos/seed/webcam4/600/400',
    ],
  },
  'SPK-BT': {
    main: 'https://picsum.photos/seed/speaker1/600/400',
    gallery: [
      'https://picsum.photos/seed/speaker1/600/400',
      'https://picsum.photos/seed/speaker2/600/400',
      'https://picsum.photos/seed/speaker3/600/400',
      'https://picsum.photos/seed/speaker4/600/400',
    ],
  },
  'HP-ANC': {
    main: 'https://picsum.photos/seed/headphone1/600/400',
    gallery: [
      'https://picsum.photos/seed/headphone1/600/400',
      'https://picsum.photos/seed/headphone2/600/400',
      'https://picsum.photos/seed/headphone3/600/400',
      'https://picsum.photos/seed/headphone4/600/400',
      'https://picsum.photos/seed/headphone5/600/400',
    ],
  },
  'STD-LAP': {
    main: 'https://picsum.photos/seed/stand1/600/400',
    gallery: [
      'https://picsum.photos/seed/stand1/600/400',
      'https://picsum.photos/seed/stand2/600/400',
      'https://picsum.photos/seed/stand3/600/400',
      'https://picsum.photos/seed/stand4/600/400',
    ],
  },
  'LMP-LED': {
    main: 'https://picsum.photos/seed/lamp1/600/400',
    gallery: [
      'https://picsum.photos/seed/lamp1/600/400',
      'https://picsum.photos/seed/lamp2/600/400',
      'https://picsum.photos/seed/lamp3/600/400',
      'https://picsum.photos/seed/lamp4/600/400',
    ],
  },
  'CHG-QI': {
    main: 'https://picsum.photos/seed/charger1/600/400',
    gallery: [
      'https://picsum.photos/seed/charger1/600/400',
      'https://picsum.photos/seed/charger2/600/400',
      'https://picsum.photos/seed/charger3/600/400',
      'https://picsum.photos/seed/charger4/600/400',
    ],
  },
  'WATCH-SM': {
    main: 'https://picsum.photos/seed/watch1/600/400',
    gallery: [
      'https://picsum.photos/seed/watch1/600/400',
      'https://picsum.photos/seed/watch2/600/400',
      'https://picsum.photos/seed/watch3/600/400',
      'https://picsum.photos/seed/watch4/600/400',
      'https://picsum.photos/seed/watch5/600/400',
    ],
  },
  'SSD-1TB': {
    main: 'https://picsum.photos/seed/ssd1/600/400',
    gallery: [
      'https://picsum.photos/seed/ssd1/600/400',
      'https://picsum.photos/seed/ssd2/600/400',
      'https://picsum.photos/seed/ssd3/600/400',
      'https://picsum.photos/seed/ssd4/600/400',
    ],
  },
  'CBL-ETH': {
    main: 'https://picsum.photos/seed/cable1/600/400',
    gallery: [
      'https://picsum.photos/seed/cable1/600/400',
      'https://picsum.photos/seed/cable2/600/400',
      'https://picsum.photos/seed/cable3/600/400',
      'https://picsum.photos/seed/cable4/600/400',
    ],
  },
  'PAD-XL': {
    main: 'https://picsum.photos/seed/mousepad1/600/400',
    gallery: [
      'https://picsum.photos/seed/mousepad1/600/400',
      'https://picsum.photos/seed/mousepad2/600/400',
      'https://picsum.photos/seed/mousepad3/600/400',
      'https://picsum.photos/seed/mousepad4/600/400',
    ],
  },
  'MIC-USB': {
    main: 'https://picsum.photos/seed/microphone1/600/400',
    gallery: [
      'https://picsum.photos/seed/microphone1/600/400',
      'https://picsum.photos/seed/microphone2/600/400',
      'https://picsum.photos/seed/microphone3/600/400',
      'https://picsum.photos/seed/microphone4/600/400',
    ],
  },
  'TAB-GFX': {
    main: 'https://picsum.photos/seed/tablet1/600/400',
    gallery: [
      'https://picsum.photos/seed/tablet1/600/400',
      'https://picsum.photos/seed/tablet2/600/400',
      'https://picsum.photos/seed/tablet3/600/400',
      'https://picsum.photos/seed/tablet4/600/400',
    ],
  },
  'CBL-KIT': {
    main: 'https://picsum.photos/seed/cablekit1/600/400',
    gallery: [
      'https://picsum.photos/seed/cablekit1/600/400',
      'https://picsum.photos/seed/cablekit2/600/400',
      'https://picsum.photos/seed/cablekit3/600/400',
      'https://picsum.photos/seed/cablekit4/600/400',
    ],
  },
  'ARM-MON': {
    main: 'https://picsum.photos/seed/monitorarm1/600/400',
    gallery: [
      'https://picsum.photos/seed/monitorarm1/600/400',
      'https://picsum.photos/seed/monitorarm2/600/400',
      'https://picsum.photos/seed/monitorarm3/600/400',
      'https://picsum.photos/seed/monitorarm4/600/400',
    ],
  },
  'RST-WRT': {
    main: 'https://picsum.photos/seed/wristrest1/600/400',
    gallery: [
      'https://picsum.photos/seed/wristrest1/600/400',
      'https://picsum.photos/seed/wristrest2/600/400',
      'https://picsum.photos/seed/wristrest3/600/400',
      'https://picsum.photos/seed/wristrest4/600/400',
    ],
  },
  'FLT-PRV': {
    main: 'https://picsum.photos/seed/privacyscreen1/600/400',
    gallery: [
      'https://picsum.photos/seed/privacyscreen1/600/400',
      'https://picsum.photos/seed/privacyscreen2/600/400',
      'https://picsum.photos/seed/privacyscreen3/600/400',
      'https://picsum.photos/seed/privacyscreen4/600/400',
    ],
  },
};

function getPrefix(sku: string): string {
  const parts = sku.split('-');
  return parts.length >= 2 ? `${parts[0]}-${parts[1]}` : sku;
}

export function getProductImages(sku: string): ProductImageSet {
  const prefix = getPrefix(sku);
  if (imagesByCategory[prefix]) {
    return imagesByCategory[prefix];
  }
  // Fallback: generate from SKU hash
  return {
    main: `https://picsum.photos/seed/${sku}/600/400`,
    gallery: [
      `https://picsum.photos/seed/${sku}a/600/400`,
      `https://picsum.photos/seed/${sku}b/600/400`,
      `https://picsum.photos/seed/${sku}c/600/400`,
      `https://picsum.photos/seed/${sku}d/600/400`,
    ],
  };
}

export function getProductMainImage(sku: string): string {
  return getProductImages(sku).main;
}
