import { useAnimations, useGLTF } from "@react-three/drei";
import { Canvas } from "@react-three/fiber";
import { Suspense, useEffect, useRef } from "react";
import type { Group } from "three";

const MODEL_BY_NAME: Record<string, { path: string; rotationY: number; scale: number }> = {
  python: { path: "/python.glb", rotationY: 0, scale: 2.5 },
  swift: { path: "/swift.glb", rotationY: Math.PI / 4 + Math.PI + 0.6, scale: 3.0 },
};

function getModel(pokemonName: string) {
  return MODEL_BY_NAME[pokemonName.toLowerCase()] ?? MODEL_BY_NAME.python;
}

function Model({ url, rotationY, scale }: { url: string; rotationY: number; scale: number }) {
  const group = useRef<Group>(null);
  const { scene, animations } = useGLTF(url);
  const { actions, names } = useAnimations(animations, group);

  useEffect(() => {
    for (const name of names) {
      actions[name]?.reset().play();
    }
    return () => {
      for (const name of names) {
        actions[name]?.stop();
      }
    };
  }, [actions, names]);

  return (
    <group ref={group} rotation={[0, rotationY, 0]} scale={[scale, scale, scale]}>
      <primitive object={scene} />
    </group>
  );
}

export function CapturePokemonModel({
  pokemonName,
  className,
}: {
  pokemonName: string;
  className?: string;
}) {
  const model = getModel(pokemonName);
  return (
    <div className={`capture-pokemon-model-wrapper${className ? ` ${className}` : ""}`}>
      <Canvas
        camera={{ position: [0, 1.5, 3.2], fov: 45, near: 0.1 }}
        style={{ pointerEvents: "none", width: "100%", height: "100%" }}
      >
        <ambientLight intensity={0.8} />
        <directionalLight position={[5, 5, 5]} intensity={1.2} />
        <directionalLight position={[-3, 2, -3]} intensity={0.4} />
        <Suspense fallback={null}>
          <Model url={model.path} rotationY={model.rotationY} scale={model.scale} />
        </Suspense>
      </Canvas>
    </div>
  );
}

export function preloadCapturePokemonModel(pokemonName: string) {
  const model = getModel(pokemonName);
  useGLTF.preload(model.path);
}
